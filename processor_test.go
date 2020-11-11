package socket

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/stretchr/testify/assert"
	"net"
	"sync"
	"testing"
	"time"
)

type testHandler struct {
	ass      *testing.T
	isClient bool
	panic    uint8
}

func (t testHandler) OnReadError(scheduler *Scheduler, err error) {

}

func (t testHandler) OnWriteError(scheduler *Scheduler, wantWrite []byte, err error) {
	assert.Nil(t.ass, err)
}

func (t *testHandler) OnWrite(scheduler *Scheduler, wantWrite []byte, writeCount int) {
	if t.isClient {
		if len(wantWrite) > 1 {
			assert.EqualValues(t.ass, wantWrite, []byte{0x07, 0x01})
		}
	} else {
		assert.EqualValues(t.ass, wantWrite, []byte{0x03, 0x00, 0x04, 0x01})
	}
}

func (t *testHandler) OnConnected(scheduler *Scheduler) {
	if !t.isClient {
		scheduler.Send([]byte{0x03, 0x00, 0x04, 0x01})
	}
}

func (t testHandler) OnClosed(scheduler *Scheduler, err error) {

}

func (t *testHandler) OnWork(b []byte, scheduler *Scheduler) {
	if t.isClient {
		one := binary.LittleEndian.Uint16(b[0:2])
		two := binary.LittleEndian.Uint16(b[2:4])
		by := make([]byte, 2)
		binary.LittleEndian.PutUint16(by, one+two)
		scheduler.Send(by)
		time.Sleep(time.Millisecond * 50)
		scheduler.Send([]byte{t.panic})
	} else {
		if len(b) == 1 {
			t.ass.Logf("panic flag: %#x", b[0])
			switch b[0] {
			case 0xff:
				panic("test error")
			case 0xfe:
				panic(errors.New("test err"))
			}
		}
	}
}

func TestProcessor(t *testing.T) {
	srv, err := net.Listen("tcp", "127.0.0.1:29999")
	assert.Nil(t, err)
	_, err = NewProcessor(nil, nil, nil, 0)
	assert.Equal(t, err, ErrNilConnection)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				srv.Close()
				return
			default:
				conn1, err := srv.Accept()
				if err == nil {
					_, err = NewProcessor(nil, conn1, nil, 0)
					assert.Equal(t, err, ErrNilHandler)
					psr1, err := NewProcessor(ctx, conn1, &testHandler{ass: t, isClient: false}, 0)
					assert.Nil(t, err)
					assert.Equal(t, psr1.workNum, defaultWorkNum)
					assert.Equal(t, cap(psr1.workChan), defaultWorkChanCap)
					assert.Equal(t, cap(psr1.Scheduler.WriteChan), defaultWriteChanCap)
					wg.Add(1)
					go func() {
						defer wg.Done()
						psr1.Process()
						t.Log(psr1.Scheduler.Conn.ReadBytes(), psr1.Scheduler.Conn.WriteBytes())
					}()
				}
			}
		}
	}()
	<-time.After(time.Millisecond * 500)
	conn2, err := net.Dial("tcp", "127.0.0.1:29999")
	assert.Nil(t, err)
	psr2, err := NewProcessor(ctx, conn2, &testHandler{ass: t, isClient: true, panic: 0xfe}, 1)
	assert.Nil(t, err)
	go func() {
		defer wg.Done()
		psr2.Process()
	}()
	conn3, err := net.Dial("tcp", "127.0.0.1:29999")
	assert.Nil(t, err)
	psr3, err := NewProcessor(ctx, conn3, &testHandler{ass: t, isClient: true, panic: 0xff}, 1)
	assert.Nil(t, err)
	go func() {
		defer wg.Done()
		go func() {
			<-time.After(time.Millisecond * 300)
			psr3.cancel()
		}()
		psr3.Process()
	}()
	conn4, err := net.Dial("tcp", "127.0.0.1:29999")
	assert.Nil(t, err)
	psr4, err := NewProcessor(ctx, conn4, &testHandler{ass: t, isClient: true, panic: 0x00}, 1)
	assert.Nil(t, err)
	go func() {
		defer wg.Done()
		go func() {
			<-time.After(time.Millisecond * 100)
			//for i := 0; i < 100; i++ {
			//	psr4.workChan <- []byte{0x07, 0x01}
			//}
			<-time.After(time.Millisecond * 500)
			psr4.cancel()
		}()
		psr4.Process()
	}()
	go func() {
		<-time.After(time.Second * 2)
		psr2.Close()
		srv.Close()
		cancel()
	}()
	wg.Wait()
	t.Log(psr2.Scheduler.Conn.ReadBytes(), psr2.Scheduler.Conn.WriteBytes())
}

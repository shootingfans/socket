package socket

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestConnection_Close(t *testing.T) {
	con1, _ := net.Pipe()
	cc := NewConnection(con1)
	assert.Equal(t, cc.enesc, int64(0))
	assert.Nil(t, cc.Close())
	e1 := cc.enesc
	assert.NotEqual(t, e1, int64(0))
	assert.Nil(t, cc.Close())
	assert.Equal(t, e1, cc.enesc)

}

func TestConnection_Created(t *testing.T) {
	con1, con2 := net.Pipe()
	cc1 := NewConnection(con1)
	time.Sleep(time.Microsecond * 10)
	cc2 := NewConnection(con2)
	assert.NotEqualValues(t, cc1.Created(), cc2.Created())
	cc2.cnesc = cc1.cnesc
	assert.EqualValues(t, cc1.Created(), cc2.Created())
}

func TestConnection_Duration(t *testing.T) {
	con1, _ := net.Pipe()
	cc1 := NewConnection(con1)
	time.Sleep(time.Microsecond * 100)
	cc1.Close()
	assert.True(t, cc1.Duration().Microseconds() >= 100)
}

func TestConnection_Read(t *testing.T) {
	con1, con2 := net.Pipe()
	cc1 := NewConnection(con1)
	buf := make([]byte, 1024)
	go func() {
		<-time.After(time.Millisecond)
		con2.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xff})
	}()
	time.Sleep(time.Millisecond * 100)
	cnt, err := cc1.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, cnt, 5)
	assert.EqualValues(t, buf[0:cnt], []byte{0x00, 0x01, 0x02, 0x03, 0xff})
}

func TestConnection_ReadBytes(t *testing.T) {
	con1, con2 := net.Pipe()
	cc1 := NewConnection(con1)
	buf := make([]byte, 1024)
	var sum int
	go func() {
		<-time.After(time.Millisecond)
		cnt, _ := con2.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xee, 0xfe})
		sum += cnt
		cnt, _ = con2.Write([]byte{0x01, 0x31, 0x42})
		sum += cnt
	}()
	time.Sleep(time.Millisecond * 100)
	_, err := cc1.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, sum, int(cc1.ReadBytes()))
	go func() {
		<-time.After(time.Millisecond)
		cnt, _ := con2.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xee, 0xfe})
		sum += cnt
		cnt, _ = con2.Write([]byte{0x01, 0x31, 0x42})
		sum += cnt
	}()
	time.Sleep(time.Millisecond * 100)
	_, err = cc1.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, sum, int(cc1.ReadBytes()))
}

func TestConnection_Write(t *testing.T) {
	con1, con2 := net.Pipe()
	cc1 := NewConnection(con1)
	buf := make([]byte, 1024)
	go func() {
		<-time.After(time.Millisecond)
		cc1.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xff})
	}()
	time.Sleep(time.Millisecond * 100)
	cnt, err := con2.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, cnt, 5)
	assert.EqualValues(t, buf[0:cnt], []byte{0x00, 0x01, 0x02, 0x03, 0xff})
}

func TestConnection_WriteBytes(t *testing.T) {
	con1, con2 := net.Pipe()
	cc1 := NewConnection(con1)
	buf := make([]byte, 1024)
	var sum int
	go func() {
		<-time.After(time.Millisecond)
		cnt, _ := cc1.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xee, 0xfe})
		sum += cnt
		cnt, _ = cc1.Write([]byte{0x01, 0x31, 0x42})
		sum += cnt
	}()
	time.Sleep(time.Millisecond * 100)
	_, err := con2.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, sum, int(cc1.WriteBytes()))
	go func() {
		<-time.After(time.Millisecond)
		cnt, _ := cc1.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xee, 0xfe})
		sum += cnt
		cnt, _ = cc1.Write([]byte{0x01, 0x31, 0x42})
		sum += cnt
	}()
	time.Sleep(time.Millisecond * 100)
	_, err = con2.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, sum, int(cc1.WriteBytes()))
}

func TestNewConnection(t *testing.T) {
	con1, _ := net.Pipe()
	cc1 := NewConnection(con1)
	assert.EqualValues(t, cc1, &Connection{
		Conn:   con1,
		rBytes: 0,
		wBytes: 0,
		cnesc:  cc1.cnesc,
		enesc:  0,
	})
}

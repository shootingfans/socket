package socket

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"
)

var (
	defaultBufferSize   = 1024 // 默认缓冲池大小
	defaultWorkChanCap  = 20   // 默认工作通道的容量
	defaultWriteChanCap = 20   // 默认写入通道的容量
	defaultWorkNum      = 1    // 默认工作协程数量
)
var (
	ErrNilConnection = errors.New("conn is nil")
	ErrNilHandler    = errors.New("handler is nil")
)

type Processor interface {
	Process() error
	Close() error
	Scheduler() *Scheduler
}

type AsyncProcessor struct {
	handler   WorkHandler
	ctx       context.Context
	cancel    context.CancelFunc
	workChan  chan []byte
	scheduler *Scheduler
	workNum   int
	writeChan chan []byte

	wg      sync.WaitGroup
	errChan chan error
}

func (psr *AsyncProcessor) Scheduler() *Scheduler {
	return psr.scheduler
}

func (psr *AsyncProcessor) Process() error {
	psr.errChan = make(chan error, 2+psr.workNum)
	psr.wg.Add(2 + psr.workNum)
	if hdl, ok := psr.handler.(ConnectedHandler); ok {
		hdl.OnConnected(psr.scheduler)
	}
	go readProcess(psr)
	go writeProcess(psr)
	for i := 0; i < psr.workNum; i++ {
		go workProcess(psr)
	}
	psr.wg.Wait()
	var err error
	select {
	case err = <-psr.errChan:
	default:
	}
	if hdl, ok := psr.handler.(ClosedHandler); ok {
		hdl.OnClosed(psr.scheduler, err)
	}
	return err
}

func readProcess(processor *AsyncProcessor) {
	defer processor.wg.Done()
	defer close(processor.workChan)
	buf := make([]byte, defaultBufferSize)
	reader := bufio.NewReaderSize(processor.scheduler.Conn, defaultBufferSize)
	for {
		select {
		case <-processor.ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			if n > 0 {
				processor.workChan <- buf[0:n]
			}
			if err != nil {
				if err != io.EOF {
					processor.errChan <- err
					if hdl, ok := processor.handler.(ErrorHandler); ok {
						hdl.OnReadError(processor.scheduler, err)
					}
				}
				return
			}
		}
	}
}

func writeProcess(processor *AsyncProcessor) {
	defer processor.scheduler.Conn.Close()
	defer processor.wg.Done()
	defer processor.cancel()
	for by := range processor.writeChan {
		cnt, err := processor.scheduler.Conn.Write(by)
		if hdl, ok := processor.handler.(FinishHandler); ok {
			hdl.OnWrite(processor.scheduler, by, cnt)
		}
		if err != nil {
			if hdl, ok := processor.handler.(ErrorHandler); ok {
				hdl.OnWriteError(processor.scheduler, by, err)
			}
			if err == io.EOF {
				return
			}
		}
	}
}

func workProcess(processor *AsyncProcessor) {
	defer processor.wg.Done()
	defer close(processor.writeChan)
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(error); ok {
				by := make([]byte, 4096)
				cnt := runtime.Stack(by, true)
				processor.errChan <- fmt.Errorf("%w\nstrack\n%s", err, string(by[0:cnt]))
			} else {
				processor.errChan <- fmt.Errorf("panic: %v", err)
			}
			processor.cancel()
		}
	}()
	for {
		select {
		case <-processor.ctx.Done():
		clean:
			for {
				select {
				case by, ok := <-processor.workChan:
					if !ok {
						break clean
					}
					processor.handler.OnWork(by, processor.scheduler)
				default:
					break clean
				}
			}
			processor.scheduler.Conn.Close()
			return
		case by, ok := <-processor.workChan:
			if !ok {
				return
			}
			processor.handler.OnWork(by, processor.scheduler)
		}
	}
}

func (psr *AsyncProcessor) Close() error {
	psr.cancel()
	return nil
}

func NewAsyncProcessor(ctx context.Context, conn net.Conn, handler WorkHandler, workNum int) (*AsyncProcessor, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	if conn == nil {
		return nil, ErrNilConnection
	}
	if handler == nil {
		return nil, ErrNilHandler
	}
	if workNum < 1 {
		workNum = defaultWorkNum
	}
	ctx, cancel := context.WithCancel(ctx)
	writeChan := make(chan []byte, defaultWriteChanCap)
	return &AsyncProcessor{
		handler:   handler,
		ctx:       ctx,
		cancel:    cancel,
		workChan:  make(chan []byte, defaultWorkChanCap),
		workNum:   workNum,
		writeChan: writeChan,
		scheduler: NewScheduler(ctx, NewConnection(conn), writeChan),
	}, nil
}

type SyncProcessor struct {
	ctx       context.Context
	cancel    context.CancelFunc
	scheduler *Scheduler
	handler   WorkHandler
	writeChan chan []byte
}

func (psr *SyncProcessor) Process() (processErr error) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(error); ok {
				by := make([]byte, 4096)
				cnt := runtime.Stack(by, true)
				processErr = fmt.Errorf("%w\nstrack\n%s", err, string(by[0:cnt]))
			} else {
				processErr = fmt.Errorf("panic: %v", err)
			}
		}
	}()
	if hdl, ok := psr.handler.(ConnectedHandler); ok {
		hdl.OnConnected(psr.scheduler)
	}
	for {
		select {
		case <-psr.ctx.Done():
			if hdl, ok := psr.handler.(ClosedHandler); ok {
				hdl.OnClosed(psr.scheduler, nil)
			}
			return nil
		case by, ok := <-psr.writeChan:
			if !ok {
				if hdl, ok := psr.handler.(ClosedHandler); ok {
					hdl.OnClosed(psr.scheduler, nil)
				}
				return nil
			}
			psr.scheduler.Conn.SetWriteDeadline(time.Now().Add(time.Second))
			cnt, err := psr.scheduler.Conn.Write(by)
			if hdl, ok := psr.handler.(FinishHandler); ok {
				hdl.OnWrite(psr.scheduler, by, cnt)
			}
			if err != nil {
				if hdl, ok := psr.handler.(ErrorHandler); ok {
					hdl.OnWriteError(psr.scheduler, by, err)
				}
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("write bytes failed: %w", err)
			}
		default:
			by := make([]byte, 1024)
			psr.scheduler.Conn.SetDeadline(time.Now().Add(time.Second))
			cnt, err := psr.scheduler.Conn.Read(by)
			if cnt > 0 {
				psr.handler.OnWork(by[0:cnt], psr.scheduler)
			}
			if err != nil {
				if err, ok := err.(*net.OpError); ok {
					if err.Timeout() || err.Temporary() {
						continue
					}
				}
				if hdl, ok := psr.handler.(ErrorHandler); ok {
					hdl.OnReadError(psr.scheduler, err)
				}
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("read bytes failed: %w", err)
			}
		}
	}
}

func (psr *SyncProcessor) Close() error {
	psr.cancel()
	return nil
}

func (psr *SyncProcessor) Scheduler() *Scheduler {
	return psr.scheduler
}

func NewSyncProcessor(ctx context.Context, conn net.Conn, handler WorkHandler) (*SyncProcessor, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	if conn == nil {
		return nil, ErrNilConnection
	}
	if handler == nil {
		return nil, ErrNilHandler
	}
	ctx, cancel := context.WithCancel(ctx)
	writeChan := make(chan []byte, defaultWriteChanCap)
	return &SyncProcessor{
		ctx:       ctx,
		cancel:    cancel,
		writeChan: writeChan,
		scheduler: NewScheduler(ctx, NewConnection(conn), writeChan),
		handler:   handler,
	}, nil
}

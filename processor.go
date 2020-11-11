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

type Processor struct {
	handler   WorkHandler
	ctx       context.Context
	cancel    context.CancelFunc
	workChan  chan []byte
	Scheduler *Scheduler
	workNum   int
	writeChan chan []byte

	wg      sync.WaitGroup
	errChan chan error
}

func (psr *Processor) Process() error {
	psr.errChan = make(chan error, 2+psr.workNum)
	psr.wg.Add(2 + psr.workNum)
	if hdl, ok := psr.handler.(ConnectedHandler); ok {
		hdl.OnConnected(psr.Scheduler)
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
		hdl.OnClosed(psr.Scheduler, err)
	}
	return err
}

func readProcess(processor *Processor) {
	defer processor.wg.Done()
	defer close(processor.workChan)
	buf := make([]byte, defaultBufferSize)
	reader := bufio.NewReaderSize(processor.Scheduler.Conn, defaultBufferSize)
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
						hdl.OnReadError(processor.Scheduler, err)
					}
				}
				return
			}
		}
	}
}

func writeProcess(processor *Processor) {
	defer processor.Scheduler.Conn.Close()
	defer processor.wg.Done()
	defer processor.cancel()
	for by := range processor.writeChan {
		cnt, err := processor.Scheduler.Conn.Write(by)
		if hdl, ok := processor.handler.(FinishHandler); ok {
			hdl.OnWrite(processor.Scheduler, by, cnt)
		}
		if err != nil {
			if hdl, ok := processor.handler.(ErrorHandler); ok {
				hdl.OnWriteError(processor.Scheduler, by, err)
			}
			if err == io.EOF {
				return
			}
		}
	}
}

func workProcess(processor *Processor) {
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
					processor.handler.OnWork(by, processor.Scheduler)
				default:
					break clean
				}
			}
			processor.Scheduler.Conn.Close()
			return
		case by, ok := <-processor.workChan:
			if !ok {
				return
			}
			processor.handler.OnWork(by, processor.Scheduler)
		}
	}
}

func (psr *Processor) Close() error {
	psr.cancel()
	return nil
}

func NewProcessor(ctx context.Context, conn net.Conn, handler WorkHandler, workNum int) (*Processor, error) {
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
	return &Processor{
		handler:   handler,
		ctx:       ctx,
		cancel:    cancel,
		workChan:  make(chan []byte, defaultWorkChanCap),
		workNum:   workNum,
		writeChan: writeChan,
		Scheduler: NewScheduler(ctx, NewConnection(conn), writeChan),
	}, nil
}

package socket

// WorkHandler
// 工作钩子，处理器会异步读取，而读取的字节会下发到工作协程去处理，此处为工作协程做处理的逻辑
type WorkHandler interface {
	OnWork(b []byte, scheduler *Scheduler)
}

// ConnectedHandler
// 连接钩子，当初次建立连接时会进行此钩子调用
type ConnectedHandler interface {
	OnConnected(scheduler *Scheduler)
}

// ClosedHandler
// 断开钩子，当连接断开后会调用此钩子
type ClosedHandler interface {
	OnClosed(scheduler *Scheduler, err error)
}

// ErrorHandler
// 错误钩子，当发生错误时去执行
type ErrorHandler interface {

	// OnReadError
	// 当读取错误时会调用
	OnReadError(scheduler *Scheduler, err error)

	// OnWriteError
	// 当写入错误时会调用，第二个参数为试图去写入的字节
	OnWriteError(scheduler *Scheduler, wantWrite []byte, err error)
}

// FinishHandler
// 完成钩子，完成时会调用
type FinishHandler interface {

	// OnWrite
	// 当写入完成时会调用，第二个参数为试图写入的字节数组，第三个参数为实际写入的长度
	OnWrite(scheduler *Scheduler, wantWrite []byte, writeCount int)
}

package socket_test

import (
	"context"
	"github.com/shootingfans/socket"
	"log"
	"net"
)

type ExampleHandler struct {
}

func (e ExampleHandler) OnWrite(scheduler *socket.Scheduler, wantWrite []byte, writeCount int) {
	// when write finish call this method
	// do something
	log.Printf("向连接 %s 写入内容 %#x 共 %d 字节", scheduler.Conn.RemoteAddr().String(), wantWrite, writeCount)
}

func (e ExampleHandler) OnReadError(scheduler *socket.Scheduler, err error) {
	// when read from connection failed call this method
	// do something
	log.Printf("从连接 %s 读取内容失败: %v\n", scheduler.Conn.RemoteAddr().String(), err)
}

func (e ExampleHandler) OnWriteError(scheduler *socket.Scheduler, wantWrite []byte, err error) {
	// when write to connection failed call this method
	// do something
	log.Printf("写入到连接 %s 内容 %#x 失败: %v\n", scheduler.Conn.RemoteAddr().String(), wantWrite, err)
}

func (e ExampleHandler) OnClosed(scheduler *socket.Scheduler, err error) {
	// when connection closed call this method
	// do something
	log.Printf("连接 %s 断开: %v\n", scheduler.Conn.RemoteAddr().String(), err)
}

func (e ExampleHandler) OnConnected(scheduler *socket.Scheduler) {
	// when connected call this method
	// do something
	log.Printf("%s 连接了\n", scheduler.Conn.RemoteAddr().String())
}

func (e ExampleHandler) OnWork(b []byte, scheduler *socket.Scheduler) {
	// do something
	log.Printf("从连接 %s 读取到内容 %#x\n", scheduler.Conn.RemoteAddr().String(), b)
	// 将接收的数据重新写入
	scheduler.Send(b)
}

func ExampleNewProcessor() {
	server, err := net.Listen("tcp", "127.0.0.1:29999")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("accept conn failed: " + err.Error())
			continue
		}
		process, err := socket.NewProcessor(
			context.Background(),
			conn,
			new(ExampleHandler),
			1,
		)
		if err != nil {
			log.Println("create process failed: " + err.Error())
			conn.Close()
			continue
		}
		go process.Process()
	}
}

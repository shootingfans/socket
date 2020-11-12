// socket
// socket is a sample package handle net.Conn
// read and write connection by sync in routines
// use worker routines handle read bytes
// 这是一个用于简单处理net.Conn的包，其读和写均采用异步，在协程中去读写
// 使用多个工作协程去处理读取到的数据
package socket

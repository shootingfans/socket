# socket

一个方便快捷的操作 `socket` 连接的包,使用多个协程来分别处理读取、处理以及写入的工作流程。

[![Build Status](https://travis-ci.org/shootingfans/socket.svg?branch=main)](https://travis-ci.org/shootingfans/socket)
[![Coverage Status](https://coveralls.io/repos/github/shootingfans/socket/badge.svg?branch=main)](https://coveralls.io/github/shootingfans/socket?branch=main)
[![codecov](https://codecov.io/gh/shootingfans/socket/branch/main/graph/badge.svg)](https://codecov.io/gh/shootingfans/socket)

## 设计理念

### 处理器 `Processor`

处理器为对一个 `net.Conn` 类型连接的一个处理器，通过一个定义的 `WorkHandler` 来创建

#### 处理流程

处理器会创建读协程、写协程以及多个工作协程，读协程读取到数据后会发送给工作协程去处理，而工作协程则可以在处理数据后通过调度器 `Scheduler` 来发送数据给写入协程，写入协程会将数据写入给连接

### 所有`Handler`类接口说明

- `WorkHandler` 此接口在读取到数据后在写入协程中调用
- `ConnectedHandler` 若实现此接口则连接刚刚接入后即调用，此时还未读取和写入任何数据
- `ClosedHandler` 若实现此接口则会在连接断开后调用
- `ErrorHandler` 若实现此接口则会在读取或写入错误时在读取和写入协程中调用对应的方法
- `FinishHandler` 若实现此接口则会在写入数据到连接成功后在写入协程中调用

### 调度器 `Scheduler`

调度器的是在每个 `Handler` 中传入，用于在这些 `Handler` 中能获取连接状态、完成写入数据、关闭连接等操作

### 连接 `Connection`

连接是对 `net.Conn` 做的一个封装，为了能记录写入、创建时间、读取、连接时长等数据
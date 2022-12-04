package shleverror

import "errors"

var (
	// ErrServerShutdown 服务器准备关闭，无法接受新连接
	ErrServerShutdown = errors.New("server is going to be shutdown")
	// ErrServerInShutdown 当服务器重复关闭时发生该错误
	ErrServerInShutdown = errors.New("server is in shutdown")
	// ErrAcceptSocket 接受新连接错误
	ErrAcceptSocket = errors.New("accept a new connection error")
	//ErrTooManyEventLoopThreads 所需的线程数过多
	ErrTooManyEventLoopThreads = errors.New("too many event-loops under LockOSThread mode")
)

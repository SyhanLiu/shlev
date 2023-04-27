package shlev

import (
	"time"
)

type Options struct {
	// TCPKeepAlive 设置tcp连接的保活时间
	TCPKeepAlive time.Duration

	// 绑定goroutine到线程，使用tls的时候要用到，或者使用cgo，或者需要对当前进行操作，或者想让事件循环更高效运行
	LockOSThread bool

	// 是否需要给socket设置SO_REUSEPORT
	ReusePort bool

	// 是否开启多核；启动CPU数量的EventLoop，会被NumEventLoop覆盖，如果设置了NumEventLoop，Multicore就无效
	Multicore bool

	// 指定EventLoop的数量
	NumEventLoop int

	// 是否需要给socket设置SO_REUSEADDR
	ReuseAddr bool

	// 可以最多读到的数据
	ReadBufferCap int

	// 可以最多写的数据
	WriteBufferCap int

	// 是否开启Nagle算法，true表示不开启，false表示开启
	TCPNoDelay bool

	// SocketRecvBuffer 设置socket读缓冲区
	SocketRecvBuffer int

	// SocketSendBuffer 设置socket写缓冲区
	SocketSendBuffer int

	// 负载均衡器
	LB LoadBalancing
}

type OptionFunc = func(*Options)

// 设置参数，返回最终的Options结构
func loadOptions(options ...OptionFunc) *Options {
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

// WithOptions 手动设置所有选项
func WithOptions(options Options) OptionFunc {
	return func(opts *Options) {
		*opts = options
	}
}

// WithMulticore 设置开启多核
func WithMulticore(multicore bool) OptionFunc {
	return func(opts *Options) {
		opts.Multicore = multicore
	}
}

// WithLockOSThread event-loops是否锁线程
func WithLockOSThread(lockOSThread bool) OptionFunc {
	return func(opts *Options) {
		opts.LockOSThread = lockOSThread
	}
}

// WithReadBufferCap 设置读缓冲区大小
func WithReadBufferCap(readBufferCap int) OptionFunc {
	return func(opts *Options) {
		opts.ReadBufferCap = readBufferCap
	}
}

// WithWriteBufferCap 设置发送缓冲区大小
func WithWriteBufferCap(writeBufferCap int) OptionFunc {
	return func(opts *Options) {
		opts.WriteBufferCap = writeBufferCap
	}
}

// WithLoadBalancing 设置负载均衡算法
func WithLoadBalancing(lb LoadBalancing) OptionFunc {
	return func(opts *Options) {
		opts.LB = lb
	}
}

// WithNumEventLoop 指定EventLoop数量
func WithNumEventLoop(numEventLoop int) OptionFunc {
	return func(opts *Options) {
		opts.NumEventLoop = numEventLoop
	}
}

// WithReusePort 设置监听套接字端口复用
func WithReusePort(reusePort bool) OptionFunc {
	return func(opts *Options) {
		opts.ReusePort = reusePort
	}
}

// WithReuseAddr 设置地址复用
func WithReuseAddr(reuseAddr bool) OptionFunc {
	return func(opts *Options) {
		opts.ReuseAddr = reuseAddr
	}
}

// WithTCPKeepAlive 设置tcp的keep-alive机制
func WithTCPKeepAlive(tcpKeepAlive time.Duration) OptionFunc {
	return func(opts *Options) {
		opts.TCPKeepAlive = tcpKeepAlive
	}
}

// WithTCPNoDelay 开启或者关闭套接字的TCP_NODELAY选项
func WithTCPNoDelay(tcpNoDelay bool) OptionFunc {
	return func(opts *Options) {
		opts.TCPNoDelay = tcpNoDelay
	}
}

// WithSocketRecvBuffer 设置套接字接收缓冲区大小
func WithSocketRecvBuffer(recvBuf int) OptionFunc {
	return func(opts *Options) {
		opts.SocketRecvBuffer = recvBuf
	}
}

// WithSocketSendBuffer 设置套接字发送缓冲区大小
func WithSocketSendBuffer(sendBuf int) OptionFunc {
	return func(opts *Options) {
		opts.SocketSendBuffer = sendBuf
	}
}

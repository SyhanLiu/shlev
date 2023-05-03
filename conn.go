package shlev

import (
	"bytes"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"shlev/internal/netpoll"
)

// Conn 封装套接字，抽象连接
type Conn struct {
	fd         int           // 文件描述符
	lnIndex    int           // 监听器的索引
	context    interface{}   // 用户定义的上下文
	remotePeer unix.Sockaddr // 远端套接字地址
	localAddr  net.Addr      // 本地地址
	remoteAddr net.Addr      // 远端地址
	loop       *EventLoop    // 所属的事件循环
	buffer     []byte        // 保存最近读出的数据
	recvBuffer *bytes.Buffer // 对端发送过来，未处理的数据
	sendBuffer *bytes.Buffer // 需要发送给对端的数据
	opened     bool          // 连接是否打开
}

func (c *Conn) Context() interface{}       { return c.context }
func (c *Conn) SetContext(ctx interface{}) { c.context = ctx }
func (c *Conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *Conn) RemoteAddr() net.Addr       { return c.remoteAddr }

// 释放tcp连接
func (c *Conn) releaseTCP() {
	c.opened = false
	c.remotePeer = nil
	c.context = nil
	c.localAddr = nil
	c.remoteAddr = nil
	c.recvBuffer = nil
	c.sendBuffer = nil
}

// 连接打开时，发送buf给对端
func (c *Conn) open(buf []byte) error {
	n, err := unix.Write(c.fd, buf)
	// 非阻塞套接字发送缓冲区满时，返回EAGAIN错误，此时将需要发送的信息存入buffer中
	if err == unix.EAGAIN {
		c.sendBuffer.Write(buf)
		return nil
	}

	if err == nil && n < len(buf) {
		c.sendBuffer.Write(buf[n:])
	}
	return err
}

// 读数据
func (c *Conn) Read(p []byte) (n int, err error) {
	if c.recvBuffer.Len() == 0 {
		n = copy(p, c.buffer)
		c.buffer = c.buffer[n:]
		if n == 0 && len(p) > 0 {
			err = io.EOF
		}
		return n, err
	}

	n, _ = c.recvBuffer.Read(p)
	if n == len(p) {
		return
	}

	m := copy(p[n:], c.buffer)
	n += m
	c.buffer = c.buffer[m:]
	return n, err
}

// 写数据
func (c *Conn) Write(data []byte) (n int, err error) {
	n = len(data)

	// 连接发送缓冲区不为0时，说明此时套接字的发送缓冲区已经满了，没有必要向套接字写。
	if c.sendBuffer.Len() != 0 {
		c.sendBuffer.Write(data)
		return n, nil
	}

	var send int
	if send, err = unix.Write(c.fd, data); err != nil {
		// 写入错误，释放内存，关闭连接
		return -1, c.loop.closeConnection(c)
	}

	// 当套接字写缓冲区写满时，写入连接的发送缓冲区
	if send < n {
		c.sendBuffer.Write(data[send:])
		// 监听写事件
		err = c.loop.netpoll.ModReadWrite(c.fd)
	}
	return n, err
}

// TODO AsyncWrite

// 创建新的tcp连接
func newTCPConn(fd int, e *EventLoop, sa unix.Sockaddr, localAddr, remoteAddr net.Addr) (c *Conn) {
	c = &Conn{
		fd:         fd,
		lnIndex:    0,
		context:    nil,
		remotePeer: sa,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		loop:       e,
		buffer:     nil,
		recvBuffer: bytes.NewBuffer(make([]byte, e.server.opts.SocketRecvBuffer)),
		sendBuffer: bytes.NewBuffer(make([]byte, e.server.opts.SocketSendBuffer)),
		opened:     false,
	}
	c.sendBuffer = bytes.NewBuffer(make([]byte, 0))
	return
}

func (c *Conn) handleEvents(_ int, ev uint32) error {
	// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
	// sure what you're doing!
	// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

	// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
	// the peer when any error occurs on a connection.
	//
	// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
	// In either case write() should take care of it properly:
	// 1) writing data back,
	// 2) closing the connection.
	if ev&netpoll.OutEvents != 0 && !(c.sendBuffer.Len() == 0) {
		if err := c.loop.write(c); err != nil {
			return err
		}
	}
	if ev&netpoll.InEvents != 0 {
		return c.loop.read(c)
	}

	return nil
}

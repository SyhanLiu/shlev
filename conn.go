package shlev

import (
	"bytes"
	"golang.org/x/sys/unix"
	"net"
	"shlev/internal/netpoll"
)

// Conn 封装套接字，抽象连接
type Conn struct {
	fd             int                     // 文件描述符
	lnIndex        int                     // 监听器的索引
	context        interface{}             // 用户定义的上下文
	remotePeer     unix.Sockaddr           // 远端套接字地址
	localAddr      net.Addr                // 本地地址
	remoteAddr     net.Addr                // 远端地址
	loop           *EventLoop              // 所属的事件循环
	buffer         []byte                  // 保存最近读出的数据
	recvBuffer     *bytes.Buffer           // 对端发送过来，未处理的数据
	sendBuffer     *bytes.Buffer           // 需要发送给对端的数据
	pollAttachment *netpoll.PollAttachment // connection attachment for poller
	opened         bool                    // 连接是否打开
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
	netpoll.PutPollAttachment(c.pollAttachment)
	c.pollAttachment = nil
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

// 创建新的tcp连接
func newTCPConn(fd int, e *EventLoop, sa unix.Sockaddr, localAddr, remoteAddr net.Addr) (c *Conn) {
	c = &Conn{
		fd:         fd,
		remotePeer: sa,
		loop:       e,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
	c.sendBuffer = bytes.NewBuffer(make([]byte, 0))
	c.pollAttachment = netpoll.GetPollAttachment()
	c.pollAttachment.FD, c.pollAttachment.Callback = fd, c.handleEvents
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

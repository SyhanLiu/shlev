package shlev

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"runtime"
	"shlev/internal/netpoll"
	"shlev/internal/socket"
	"shlev/tools/logger"
	"shlev/tools/shleverror"
	"sync/atomic"
	"time"
)

type EventLoop struct {
	ln     *Listener // 监听的套接字
	index  int       // 该指针[]*EventLoop中的索引，事件循环列表中的索引
	cache  bytes.Buffer
	server *Server // 所属的server

	buffer           []byte            // 缓冲区
	tcpConnectionMap map[int]*Conn     // key：fd， value：Conn
	connCount        int32             // 活跃连接数，连接需要时打开状态（opened=true）
	netpoll          netpoll.Netpoller // 可循环对象（epoll）
	eventHandler     EventHandler      // 用户定义的事件、连接钩子回调
}

func (e *EventLoop) addConn(delta int32) {
	atomic.AddInt32(&e.connCount, delta)
}

func (e *EventLoop) loadConn() int32 {
	return atomic.LoadInt32(&e.connCount)
}

func (e *EventLoop) closeAllConnections() {
	for _, c := range e.tcpConnectionMap {
		_ = e.closeConnection(c)
	}
}

func (e *EventLoop) closeConnection(c *Conn) (err error) {
	// 连接关闭时，直接返回
	if !c.opened {
		return
	}
	// 连接
	if addr := c.localAddr; addr != nil {
		return
	}

	// 如果发送缓冲不为空，说明还有数据要发送，需要先发送完数据再关闭连接
	if c.sendBuffer.Len() != 0 {
		n, err := unix.Write(c.fd, c.sendBuffer.Bytes())
		if err != nil {
			logger.Error(fmt.Sprintf("closeConnection fd:%d error:%v", c.fd, err))
		} else {
			c.sendBuffer.Truncate(n)
		}
	}

	// 从netpoll删除fd
	err0 := e.netpoll.Delete(c.fd)
	if err0 != nil {
		err = fmt.Errorf("failed to delete fd=%d from netpoll in eventloop(%d):%v", c.fd, e.index, err0)
		logger.Error(err)
	}
	// 关闭连接
	err1 := unix.Close(c.fd)
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d from netpoll in eventloop(%d):%v", c.fd, e.index, err1)
		logger.Error(err1)
		if err != nil {
			err = errors.New(err.Error() + "&&" + err1.Error())
		} else {
			err = err1
		}
	}

	delete(e.tcpConnectionMap, c.fd)
	e.addConn(-1)
	e.eventHandler.OnConnectionClose(c, err)
	c.releaseTCP()
	return err
}

// Register 给连接注册事件
func (e *EventLoop) Register(c *Conn) error {
	if err := e.netpoll.AddRead(c.fd); err != nil {
		_ = unix.Close(c.fd)
		c.releaseTCP()
		return err
	}
	e.tcpConnectionMap[c.fd] = c
	return e.open(c)
}

func (e *EventLoop) open(c *Conn) error {
	c.opened = true
	e.addConn(1)

	buf, result := e.eventHandler.OnOpen(c, nil)

	if err := c.open(buf); err != nil {
		return err
	}

	if c.sendBuffer.Len() != 0 {
		if err := e.netpoll.AddWrite(c.fd); err != nil {
			return err
		}
	}

	return e.handleResult(c, result)
}

func (e *EventLoop) read(c *Conn) error {
	n, err := unix.Read(c.fd, e.buffer)
	if err == nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			err = unix.ECONNRESET
		}
		logger.Error(fmt.Sprintf("EventLoop event_loop idx:%d read err:%v", c.fd, os.NewSyscallError("read", err)))
		return e.closeConnection(c)
	}

	c.buffer = e.buffer[:n]
	result := e.eventHandler.OnTraffic(c)
	switch result {
	case None:
	case Close:
		return e.closeConnection(c)
	case Shutdown:
		return shleverror.ErrServerShutdown
	}
	c.recvBuffer.Write(c.buffer)

	return nil
}

func (e *EventLoop) write(c *Conn) error {
	msg := c.sendBuffer.Bytes()
	n, err := unix.Write(c.fd, msg)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		} else {
			logger.Error(fmt.Sprintf("EventLoop event_loop idx:%d read err:%v", c.fd, os.NewSyscallError("write", err)))
			return e.closeConnection(c)
		}
	}

	if n > 0 {
		c.sendBuffer.Truncate(n)
	}
	// 当所有数据都发送出去时，此时没有必要继续监听写事件了
	if c.sendBuffer.Len() == 0 {
		e.netpoll.ModRead(c.fd)
	}

	return nil
}

// 唤醒连接
func (e *EventLoop) wake(c *Conn) error {
	if conn, ok := e.tcpConnectionMap[c.fd]; !ok || conn != c {
		// 忽略未更新的连接
		return nil
	}

	res := e.eventHandler.OnTraffic(c)

	return e.handleResult(c, res)
}

// todo 暂时未加入定时器功能
func (e *EventLoop) ticker(ctx context.Context) {
	if e == nil {
		return
	}

	var timer *time.Ticker
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	//for {
	//	delay, action := e.eventCallback.OnTick()
	//
	//}
}

func (e *EventLoop) handleResult(c *Conn, res HandleResult) error {
	switch res {
	case None:
		return nil
	case Close:
		return e.closeConnection(c)
	case Shutdown:
		return shleverror.ErrServerShutdown
	default:
		return nil
	}
	return nil
}

// 添加新连接
func (e *EventLoop) accept(fd int, _ uint32) error {
	connFd, sa, err := unix.Accept(fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		logger.Error(fmt.Sprintf("Accept() failed due to error: %v", err))
		return os.NewSyscallError("accept", err)
	}

	// 给新连接设置非阻塞
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(connFd, true)); err != nil {
		return err
	}

	remoteAddr := socket.SockaddrToTCPAddr(sa)
	if e.server.opts.TCPKeepAlive > 0 {
		err = socket.SetKeepAlivePeriod(connFd, int(e.server.opts.TCPKeepAlive/time.Second))
		logger.Error(err)
	}

	c := newTCPConn(connFd, e, sa, e.ln.Addr, remoteAddr)
	if err = e.netpoll.AddRead(c.fd); err != nil {
		return err
	}
	e.tcpConnectionMap[c.fd] = c
	return e.open(c)
}

// 开启当前事件循环
func (e *EventLoop) run(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() {
		e.closeAllConnections()
		e.ln.Close()
		e.server.signalShutdown()
	}()

	// 即负责io也负责accept
	err := e.netpoll.Polling(func(fd int, ev uint32) error {
		if c, ok := e.tcpConnectionMap[fd]; ok {
			// 如果对方挂断，在write函数中会处理rdhup和hup事件
			// 无论是否有错误，都要把发送缓冲区的数据发送完毕之后才关闭连接
			// 发生错误时，write要保证两点：1、发送完待发送数据；2、关闭连接
			if (ev&netpoll.OutEvents) != 0 && !(c.sendBuffer.Len() == 0) {
				if err := e.write(c); err != nil {
					return err
				}
			}
			if (ev & netpoll.InEvents) != 0 {
				return e.read(c)
			}
			return nil
		}
		return e.accept(fd, ev)
	})

	if err != nil {
		logger.Error(fmt.Sprintf("event-loop(%d) is exiting due to error: %v", e.index, err))
	}
}

func (e *EventLoop) activateSubReactor(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() {
		e.closeAllConnections()
		e.server.signalShutdown()
	}()

	// 从reactor只需要处理i/o
	err := e.netpoll.Polling(func(fd int, ev uint32) error {
		if c, ok := e.tcpConnectionMap[fd]; ok {
			// 如果对方挂断，在write函数中会处理rdhup和hup事件
			// 无论是否有错误，都要把发送缓冲区的数据发送完毕之后才关闭连接
			// 发生错误时，write要保证两点：1、发送完待发送数据；2、关闭连接
			if (ev&netpoll.OutEvents) != 0 && !(c.sendBuffer.Len() == 0) {
				if err := e.write(c); err != nil {
					return err
				}
			}
			if (ev & netpoll.InEvents) != 0 {
				return e.read(c)
			}
		}
		return nil
	})

	if err != nil {
		logger.Debug(fmt.Sprintf("event-loop(%d) is exiting due to error: %v", e.index, err))
	}
}

func (e *EventLoop) activateMainReactor(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer e.server.signalShutdown()

	// 主reactor只负责accept
	err := e.netpoll.Polling(func(fd int, ev uint32) error { return e.server.accept(fd, ev) })
	if err == shleverror.ErrServerShutdown {
		logger.Error("main reactor is exiting in terms of the demand from user, error:", err)
	} else if err != nil {
		logger.Error("main reactor is exiting due to error:", err)
	}
}

// 事件循环注册函数
func (e *EventLoop) register(itf interface{}) error {
	c := itf.(*Conn)

	if err := e.netpoll.AddRead(c.fd); err != nil {
		_ = unix.Close(c.fd)
		c.releaseTCP()
		return err
	}
	e.tcpConnectionMap[c.fd] = c
	return e.open(c)
}

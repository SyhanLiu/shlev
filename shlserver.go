package shlev

import (
	"bytes"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"runtime"
	"shlev/internal/netpoll"
	"shlev/internal/socket"
	"shlev/tools/logger"
	"shlev/tools/shleverror"
	"sync"
	"sync/atomic"
)

type Server struct {
	ln           *Listener      // 监听器，监听端口建立连接
	lb           loadBalancer   // 负载均衡算法
	wg           sync.WaitGroup // 表示有多少eventLoop开启，关闭server需要等开启的eventLoop关闭
	once         sync.Once      // 确保signalShutdown只关闭一次
	cond         *sync.Cond     // 处理服务器关闭的信号
	mainLoop     *EventLoop     // 主事件循环，接收连接
	inShutdown   int32          // 1：正在关闭server
	opts         *Options       // 可设置选项
	eventHandler EventHandler   // 事件处理handler
}

// server是否正在关闭中
func (s *Server) isInShutdown() bool {
	return atomic.LoadInt32(&s.inShutdown) == 1
}

// 等待信号关闭server
func (s *Server) waitForShutdown() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

// 发送信号让server关闭
func (s *Server) signalShutdown() {
	s.once.Do(func() {
		s.cond.L.Lock()
		s.cond.Signal()
		s.cond.L.Unlock()
	})
}

// 开始事件循环
func (s *Server) startEventLoops() {
	s.lb.iterate(func(i int, e *EventLoop) bool {
		s.wg.Add(1)
		go func() {
			// 锁线程，获取更高效的性能
			e.run(s.opts.LockOSThread)
			s.wg.Done()
		}()
		return true
	})
}

// 关闭事件循环
func (s *Server) closeEventLoops() {
	s.lb.iterate(func(i int, e *EventLoop) bool {
		e.netpoll.Close()
		return true
	})
}

// 激活事件循环
func (s *Server) activateEventLoops(numEventLoop int) (err error) {
	address := s.ln.Address
	ln := s.ln
	s.ln = nil
	// 创建EventLoop并且绑定Listener
	for i := 0; i < numEventLoop; i++ {
		if i > 0 {
			if ln, err = NewTCP4Listener(address, s.opts); err != nil {
				return
			}
		}

		var p netpoll.Netpoller = netpoll.NewEpoller()
		if err = p.Init(); err == nil {
			el := new(EventLoop)
			el.ln = ln
			el.server = s
			el.netpoll = p
			el.buffer = make([]byte, s.opts.ReadBufferCap)
			el.tcpConnectionMap = make(map[int]*Conn)
			el.eventHandler = s.eventHandler
			el.ln.SetAcceptCallback(el.accept)
			if err = el.netpoll.AddRead(el.ln.Fd); err != nil {
				return
			}
			s.lb.register(el)
		} else {
			return
		}
	}

	// 开始后台运行事件循环
	s.startEventLoops()

	return
}

// 激活响应器
func (s *Server) activateReactors(numEventLoop int) error {
	for i := 0; i < numEventLoop; i++ {
		var p netpoll.Netpoller = netpoll.NewEpoller()
		if err := p.Init(); err == nil {
			el := &EventLoop{
				ln:               s.ln,
				index:            0,
				cache:            bytes.Buffer{},
				server:           s,
				buffer:           make([]byte, s.opts.ReadBufferCap),
				tcpConnectionMap: make(map[int]*Conn),
				netpoll:          p,
				eventHandler:     s.eventHandler,
			}
			s.lb.register(el)
		} else {
			return err
		}
	}

	// 开启从响应器
	s.startSubReactors()

	// 建立主响应器，主响应器只负责监听端口建立连接
	var p netpoll.Netpoller = netpoll.NewEpoller()
	if err := p.Init(); err == nil {
		e := &EventLoop{
			ln:           s.ln,
			index:        -1,
			server:       s,
			netpoll:      p,
			eventHandler: s.eventHandler,
		}
		if err = e.netpoll.AddRead(e.ln.Fd); err != nil {
			return err
		}
		// 设置主响应器指针
		s.mainLoop = e
		// 开启主响应器
		s.wg.Add(1)
		go func() {
			e.activateMainReactor(s.opts.LockOSThread)
			s.wg.Done()
		}()
	} else {
		return err
	}
	return nil
}

func (s *Server) startSubReactors() {
	s.lb.iterate(func(i int, el *EventLoop) bool {
		s.wg.Add(1)
		go func() {
			el.activateSubReactor(s.opts.LockOSThread)
			s.wg.Done()
		}()
		return true
	})
}

// 开启事件循环
func (s *Server) start(numEventLoop int) error {
	if s.opts.ReusePort {
		// 类nginx
		// 使用端口复用模式开启事件循环，多个线程监听同一个端口，每个线程都负责accpet，read，write
		return s.activateEventLoops(numEventLoop)
	}

	// 类redis
	// 开启一主多从reactor模式，主负责accept，从负责read，write
	return s.activateReactors(numEventLoop)
}

// 开启服务器
func serve(eventHandler EventHandler, listener *Listener, options *Options, addr string) error {
	// 计算EventLoop数量
	numEventLoop := 1
	if options.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if options.NumEventLoop > 0 {
		numEventLoop = options.NumEventLoop
	}

	s := &Server{
		ln:           listener,
		opts:         options,
		eventHandler: eventHandler,
	}

	// 根据负载均衡枚举值设置负载均衡器
	switch options.LB {
	case RoundRobin:
		s.lb = &roundRobinLoadBalancer{}
	case LeastConnections:
		s.lb = &leastConnectionsLoadBalancer{}
	case SourceAddrHash:
		s.lb = &sourceAddrHashLoadBalancer{}
	}

	s.cond = sync.NewCond(&sync.Mutex{})
	// 执行启动钩子函数
	err := s.eventHandler.OnBoot(s)
	if err != nil {
		logger.Error("server OnBoot error:", err)
		return err
	}

	if err = s.start(numEventLoop); err != nil {
		s.closeEventLoops()
		logger.Error("server start error:", err)
		return err
	}
	defer s.stop()

	allServers.Store(addr, s)
	return nil
}

// 在主从响应器模式中使用，并且只会由主响应器调用
func (s *Server) accept(fd int, _ uint32) error {
	nfd, sa, err := unix.Accept(fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		logger.Error("Accept() failed due to error:", err)
		return shleverror.ErrAcceptSocket
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	remoteAddr := socket.SockaddrToTCPAddr(sa)
	if s.opts.TCPKeepAlive > 0 {
		err = socket.SetKeepAlivePeriod(nfd, int(s.opts.TCPKeepAlive.Seconds()))
		logger.Error("set keep-alive error:", err)
	}

	el := s.lb.next(remoteAddr)
	c := newTCPConn(nfd, el, sa, el.ln.Addr, remoteAddr)

	err = el.netpoll.AddUrgentTask(el.register, c)
	if err != nil {
		logger.Error(fmt.Sprintf("AddUrgentTask failed due to error: %v", err))
		_ = unix.Close(nfd)
		c.releaseTCP()
	}
	return nil
}

func (s *Server) stop() {
	// 等待信号进行关闭
	s.waitForShutdown()

	// 执行关闭服务器时的钩子函数
	s.eventHandler.OnShutdown(s)

	// 通知所有的loop关闭所有listener
	s.lb.iterate(func(i int, e *EventLoop) bool {
		err := e.netpoll.AddUrgentTask(func(_ interface{}) error { return shleverror.ErrServerShutdown }, nil)
		if err != nil {
			logger.Error("failed to call AddUrgentTask on sub event-loop when stopping engine:", err)
		}
		return true
	})

	// 主从reactor模式下，只需要关闭主reactor
	if s.mainLoop != nil {
		s.ln.Close()
		err := s.mainLoop.netpoll.AddUrgentTask(func(_ interface{}) error { return shleverror.ErrServerShutdown }, nil)
		if err != nil {
			logger.Error("failed to call AddUrgentTask on main event-loop when stopping engine:", err)
		}
	}

	// 等待所有循环完成读取事件
	s.wg.Wait()

	s.closeEventLoops()

	if s.mainLoop != nil {
		err := s.mainLoop.netpoll.Close()
		if err != nil {
			logger.Error("failed to close poller when stopping engine:", err)
		}
	}

	atomic.StoreInt32(&s.inShutdown, 1)
}

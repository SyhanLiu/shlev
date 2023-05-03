package shlev

import (
	"context"
	"fmt"
	"github.com/Senhnn/shlev/tools/logger"
	"github.com/Senhnn/shlev/tools/shleverror"
	"sync"
	"time"
)

// MaxTcpBufferCap tcp读/写缓冲区的最大值
const MaxTcpBufferCap = 64 * 1024 // 64KB

type HandleResult = int

const (
	// None 在事件之后不需要做任何操作
	None HandleResult = iota

	// Close 事件之后应该关闭连接
	Close

	// Shutdown 停止服务器
	Shutdown
)

// EventHandler 事件循环钩子回调
type EventHandler interface {
	// OnBoot 当服务器开启时触发
	OnBoot(*Server) error

	// OnShutdown 当服务器关闭时会调用，他会关闭所有的事件循环和连接
	OnShutdown(*Server)

	// OnConnectionClose 在连接关闭时触发钩子
	OnConnectionClose(*Conn, error)

	// OnOpen 连接打开时触发钩子
	OnOpen(*Conn, error) ([]byte, HandleResult)

	// OnTraffic 当套接字收到数据时触发
	OnTraffic(*Conn) HandleResult
}

var allServers sync.Map

func Run(eventHandler EventHandler, addr string, opts ...OptionFunc) error {
	// 整理选项参数
	options := loadOptions(opts...)

	if options.LockOSThread && options.NumEventLoop > 10000 {
		logger.Error(fmt.Sprintf("too many event-loops under LockOSThread mode, should be less than 10,000 "+
			"while you are trying to set up %d\n", options.NumEventLoop))
		return shleverror.ErrTooManyEventLoopThreads
	}

	// 目前写死，能跑了之后在加功能，64K
	options.ReadBufferCap = MaxTcpBufferCap
	options.WriteBufferCap = MaxTcpBufferCap

	var l *Listener
	var err error
	if l, err = NewTCP4Listener(addr, options); err != nil {
		logger.Error("Run err:", err)
		return err
	}
	defer l.Close()

	return serve(eventHandler, l, options, addr)
}

// Stop 优雅关闭服务器且不中断任何活跃的事件循环，直到连接和事件循环都关闭，最后关闭服务器
func Stop(ctx context.Context, addr string) error {
	var eng *Server
	if s, ok := allServers.Load(addr); ok {
		eng = s.(*Server)
		eng.signalShutdown()
		defer allServers.Delete(addr)
	} else {
		return shleverror.ErrServerInShutdown
	}

	if eng.isInShutdown() {
		return shleverror.ErrServerInShutdown
	}

	// 每一秒tick一次
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if eng.isInShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

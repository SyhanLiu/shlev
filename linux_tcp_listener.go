package shlev

import (
	"fmt"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"shlev/internal/socket"
	"shlev/tools/logger"
	"sync"
)

type Listener struct {
	once             sync.Once
	Fd               int
	Addr             net.Addr
	Address, Network string
	SockOpts         []socket.SocketOption
	//pollAttachment   *netpoll.PollAttachment // listener attachment for poller
	acceptCallback func(int, uint32) error
}

func ConvertOptionToSocketOption(options *Options) ([]socket.SocketOption, error) {
	var sockOpts []socket.SocketOption

	if options.ReusePort {
		sockOpt := socket.SocketOption{SetSockOpt: socket.SetReusePort, Opt: 1}
		sockOpts = append(sockOpts, sockOpt)
	}
	if options.ReuseAddr {
		sockOpt := socket.SocketOption{SetSockOpt: socket.SetReuseAddr, Opt: 1}
		sockOpts = append(sockOpts, sockOpt)
	}
	if options.TCPNoDelay {
		sockOpt := socket.SocketOption{SetSockOpt: socket.SetNoDelay, Opt: 1}
		sockOpts = append(sockOpts, sockOpt)
	}
	if options.SocketRecvBuffer > 0 {
		sockOpt := socket.SocketOption{SetSockOpt: socket.SetRecvBuffer, Opt: options.SocketRecvBuffer}
		sockOpts = append(sockOpts, sockOpt)
	}
	if options.SocketSendBuffer > 0 {
		sockOpt := socket.SocketOption{SetSockOpt: socket.SetSendBuffer, Opt: options.SocketSendBuffer}
		sockOpts = append(sockOpts, sockOpt)
	}
	return sockOpts, nil
}

func NewTCP4Listener(addr string, options *Options) (l *Listener, err error) {
	socketOpts, err := ConvertOptionToSocketOption(options)
	if err != nil {
		logger.Error("NewTCP4Listener error:", err)
		return nil, err
	}
	l = &Listener{
		once:     sync.Once{},
		Fd:       0,
		Addr:     nil,
		Address:  addr,
		Network:  "tcp",
		SockOpts: socketOpts,
	}
	l.Fd, l.Addr, err = socket.TCP4ListenSocket(addr)
	if err != nil {
		logger.Error(fmt.Sprintf("NewTCP4Listener create new listener addr:%s, error: %s", addr, err))
	}
	return l, err
}

func (l *Listener) SetAcceptCallback(f func(int, uint32) error) {
	l.acceptCallback = f
}

func (l *Listener) GetFD() int {
	return l.Fd
}

func (l *Listener) Close() {
	l.once.Do(func() {
		if l.Fd > 0 {
			err := unix.Close(l.Fd)
			if err != nil {
				logger.Error(os.NewSyscallError("close", err))
			} else {
				logger.Debug("ln close success!")
			}
		}
	})
}

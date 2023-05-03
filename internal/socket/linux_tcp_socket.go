package socket

import (
	"bufio"
	"errors"
	"github.com/Senhnn/shlev/tools/logger"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"strconv"
	"strings"
)

type FD = int

var ipv4InIPv6Prefix = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}

// 监听端口的连接队列和半连接队列长度
var listenerBacklogMaxSize = 128

// ListenerBacklogMaxSize 获取服务器配置
func ListenerBacklogMaxSize() int {
	fd, err := os.Open("/proc/sys/net/core/somaxconn")
	if err != nil {
		return unix.SOMAXCONN
	}
	defer fd.Close()

	rd := bufio.NewReader(fd)
	line, err := rd.ReadString('\n')
	if err != nil {
		return unix.SOMAXCONN
	}

	f := strings.Fields(line)
	if len(f) < 1 {
		return unix.SOMAXCONN
	}

	n, err := strconv.Atoi(f[0])
	if err != nil || n == 0 {
		return unix.SOMAXCONN
	}
	return n
}

// SocketOption 设置套接字选项
type SocketOption struct {
	SetSockOpt func(int, int) error
	Opt        int
}

// TCP4ListenSocket 新建一个监听套接字
func TCP4ListenSocket(addr string, sockOpts ...SocketOption) (fd FD, netAddr net.Addr, err error) {
	sa, netAddr, err := GetTCP4SockAddr(addr)
	if err != nil {
		logger.Error(err)
		return
	}

	if fd, err = unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP); err != nil {
		err = os.NewSyscallError("socket", err)
		logger.Error(err)
		return
	}

	for _, sockOpt := range sockOpts {
		if err = sockOpt.SetSockOpt(fd, sockOpt.Opt); err != nil {
			return
		}
	}

	// 绑定套接字
	if err = os.NewSyscallError("bind", unix.Bind(fd, sa)); err != nil {
		logger.Error(err)
		return
	}

	// 设置backlog
	err = os.NewSyscallError("listen", unix.Listen(fd, listenerBacklogMaxSize))

	return fd, netAddr, err
}

// GetTCP4SockAddr 获得出IPV4,TCP套接字的地址
func GetTCP4SockAddr(addr string) (sa *unix.SockaddrInet4, tcpAddr *net.TCPAddr, err error) {
	// 解析地址并返回对应结构
	tcpAddr, err = net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logger.Error(err)
		return
	}

	if len(tcpAddr.IP) == 0 {
		tcpAddr.IP = net.IPv4zero
	}
	ip4 := tcpAddr.IP.To4()
	if ip4 == nil {
		return &unix.SockaddrInet4{}, tcpAddr, &net.AddrError{Err: "non-IPv4 address", Addr: tcpAddr.IP.String()}
	}

	addr4 := &unix.SockaddrInet4{Port: tcpAddr.Port}
	copy(addr4.Addr[:], ip4)

	return addr4, tcpAddr, nil
}

// SockaddrToTCPAddr 把SockAddr转换为TCPAddr
func SockaddrToTCPAddr(sa unix.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		ip := sockaddrInet4ToIP(sa)
		return &net.TCPAddr{IP: ip, Port: sa.Port}
	}
	return nil
}

// 把SockaddrInet4转换成net.IP
func sockaddrInet4ToIP(sa *unix.SockaddrInet4) net.IP {
	ip := make([]byte, 16)
	copy(ip[0:12], ipv4InIPv6Prefix)
	copy(ip[12:16], sa.Addr[:])
	return ip
}

// SetKeepAlivePeriod 设置长连接keep-alive
func SetKeepAlivePeriod(fd, secs int) error {
	if secs <= 0 {
		return errors.New("invalid time duration")
	}
	// 开启keepalive机制
	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)); err != nil {
		return err
	}
	// 在tcp_keepalive_time之后，没有接收到对方确认，继续发送保活探测包的发送频率，默认值为75s
	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs)); err != nil {
		return err
	}
	// 最后一次数据交换到TCP发送第一个保活探测包的间隔，即允许的持续空闲时长，或者说每次正常发送心跳的周期，默认值为7200s（2h）
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, secs))
}

// SetNoDelay 是否开启nagel算法，如果要提高吞吐量，则设置noDelay=0，如果要强调数据的实时性，则设置noDelay=1
func SetNoDelay(fd, noDelay int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, noDelay))
}

// SetRecvBuffer 设置套接字的接收缓冲区
func SetRecvBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, size)
}

// SetSendBuffer 设置套接字的发送缓冲区
func SetSendBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, size)
}

// SetReusePort 开启端口复用，多个监听套接字可以绑定同一个端口
func SetReusePort(fd, reusePort int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, reusePort))
}

// SetReuseAddr 开启地址复用，在time_wait等待期间依然可以监听地址端口
func SetReuseAddr(fd, reuseAddr int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, reuseAddr))
}

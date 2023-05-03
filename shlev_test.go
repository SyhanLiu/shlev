package shlev

import (
	"context"
	"fmt"
	"github.com/Senhnn/shlev/tools/logger"
	"golang.org/x/sys/unix"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s := &testServer{}
	fmt.Println("server run")
	go func() {
		fmt.Println("server after run")
		time.Sleep(time.Second * 30)
		fmt.Println("server close")
		Stop(context.Background(), "127.0.0.1:10001")
	}()
	Run(s, "127.0.0.1:10001", WithNumEventLoop(3), WithLoadBalancing(RoundRobin))
}

type testServer struct {
}

func (s *testServer) OnShutdown(*Server) {

}

func (s *testServer) OnBoot(eng *Server) error {
	//s.eng = eng
	return nil
}

func (s *testServer) OnOpen(c *Conn, err error) (b []byte, e HandleResult) {
	c.SetContext(c)
	logger.Debug("OnOpen localAddr:", c.LocalAddr(), "; remoteAddr:", c.RemoteAddr())
	unix.Write(c.fd, []byte("fuck off\n"))
	return []byte{}, None
}

func (s *testServer) OnConnectionClose(c *Conn, _ error) {
	if c.recvBuffer.Len() != 0 {
		b := c.recvBuffer.Bytes()
		fmt.Println(string(b))
	}
	//logger.Debug("OnConnectionClose localAddr:", c.LocalAddr(), "; remoteAddr:", c.RemoteAddr())
	return
}

func (s *testServer) OnTraffic(c *Conn) HandleResult {
	b := make([]byte, 100000)
	n, err := c.Read(b)
	if err != nil {
		return 0
	}
	fmt.Println("read data:", string(b[:n]))
	return None
}

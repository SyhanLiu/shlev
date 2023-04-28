package shlev

import (
	"fmt"
	"golang.org/x/sys/unix"
	"shlev/tools/logger"
	"testing"
)

func TestServer(t *testing.T) {
	s := &testServer{}
	fmt.Println("server run")
	Run(s, "47.103.116.215:10001", WithNumEventLoop(3), WithLoadBalancing(RoundRobin))
	for true {

	}
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
	unix.Close(c.fd)
	return []byte{}, None
}

func (s *testServer) OnConnectionClose(_ *Conn, _ error) {
	//s.eng = eng
	return
}
func (s *testServer) OnTraffic(_ *Conn) HandleResult {
	return None
}

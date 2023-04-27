package shlev

import (
	"fmt"
	"testing"
)

func TestServer(t *testing.T) {
	s := &testServer{}
	Run(s, "127.0.0.1:9090", WithLoadBalancing(RoundRobin))
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
	fmt.Println("OnOpen localAddr:", c.LocalAddr(), "; remoteAddr:", c.RemoteAddr())
	return
}

func (s *testServer) OnConnectionClose(_ *Conn, _ error) {
	//s.eng = eng
	return
}
func (s *testServer) OnTraffic(_ *Conn) HandleResult {
	return None
}

package shlev

import (
	"fmt"
	"testing"
)

func TestServer(t *testing.T) {
	//Run(&testServer{}, "1", nil)
}

type testServer struct {
}

func (s *testServer) OnShutdown(*Server) {

}

func (s *testServer) OnBoot(eng *Server) error {
	//s.eng = eng
	return nil
}

func (s *testServer) OnOpen(c *Conn, err error) (b []byte, e error) {
	c.SetContext(c)
	fmt.Println("OnOpen localAddr:", c.LocalAddr(), "; remoteAddr:", c.RemoteAddr())
	return
}

func (s *testServer) OnConnectionClose(_ *Conn, _ error) {
	//s.eng = eng
	return
}
func (s *testServer) OnTraffic(_ *Conn) error {
	return nil
}

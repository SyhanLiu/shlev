package netpoll

import (
	"github.com/Senhnn/shlev/tools/logger"
	"golang.org/x/sys/unix"
)

type Selecter struct {
	r       *unix.FdSet
	w       *unix.FdSet
	e       *unix.FdSet
	timeout *unix.Timeval
	fdNum   int
}

func NewSelecter(timeout *unix.Timeval) *Selecter {
	s := &Selecter{
		r:       &unix.FdSet{},
		w:       &unix.FdSet{},
		e:       &unix.FdSet{},
		timeout: timeout,
	}
	s.r.Zero()
	s.w.Zero()
	s.e.Zero()
	return s
}

func (s *Selecter) Poll() ([]int, error) {
	r := *s.r
	w := *s.w
	e := *s.e
	n, err := unix.Select(s.fdNum, &r, &w, &e, s.timeout)
	if err != nil {
		logger.Error("selecter poll error!")
		return nil, err
	}
	if n > 0 {

	}

	return nil, err
}

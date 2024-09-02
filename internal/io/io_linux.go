package io

import "golang.org/x/sys/unix"

// WriteVec 封装writev接口
func WriteVec(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return unix.Writev(fd, iov)
}

// ReadVec 封装readv接口
func ReadVec(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return unix.Readv(fd, iov)
}

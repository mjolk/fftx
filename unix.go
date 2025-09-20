//go:build linux

package main

import (
	"log"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func sender(n int) *net.Dialer {
	return &net.Dialer{
		Control: func(network, address string, conn syscall.RawConn) error {
			var operr error
			if err := conn.Control(func(fd uintptr) {
				operr = syscall.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, n)
			}); err != nil {
				return err
			}
			return operr
		},
	}
}

func listen(n int) *net.ListenConfig {
	return &net.ListenConfig{
		Control: func(network, address string, conn syscall.RawConn) error {
			var operr error
			if err := conn.Control(func(fd uintptr) {
				operr = syscall.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, n)
			}); err != nil {
				log.Fatalf("could not set sock opt")
				return err
			}

			return operr
		},
	}
}

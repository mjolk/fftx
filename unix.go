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
				log.Printf(" set sender buffer to: %d \n", n)
				size := 0
				operr = syscall.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, n)
				log.Printf("err setting sys buf: %d \n", operr)
				size, operr = syscall.GetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF)
				log.Printf("sender buffer: %d \n", size)
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

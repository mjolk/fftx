package main

import (
	"errors"
	"log"
	"net"
	"os"
	"syscall"
)

func sender(n int) *net.Dialer {
	return &net.Dialer{
		Control: func(network, address string, conn syscall.RawConn) error {
			var operr error
			if err := conn.Control(func(fd uintptr) {
				operr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, n)
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
				operr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, n)
			}); err != nil {
				log.Fatalf("could not set sock opt")
				return err
			}

			return operr
		},
	}
}

func openReadFile(path string) (*os.File, error) {
	var err error
	var file *os.File
	file, err = os.OpenFile(path, 0, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func openOrCreateFile(path string) (*os.File, error) {
	var err error
	var file *os.File
	file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.Create(path)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	return file, nil
}

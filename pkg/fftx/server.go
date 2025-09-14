package fftx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

type Server interface {
	Recv(context.Context)
	Start() error
}

type udpServer struct {
	file   *os.File
	ip     string
	port   int
	sender net.Conn
}

func (s *udpServer) recv() error {
	if s.sender == nil {
		return errors.New("No sender connected")
	}

	log.Printf("trying to copy data %d\n", 1)
	recvd, err := io.Copy(s.file, s.sender)
	if err != nil {
		return err
	}

	log.Printf("copied data %d\n", recvd)
	return nil
}

func (s *udpServer) Recv(ctx context.Context) {
	s.recv()
}

func (s *udpServer) Start() error {
	log.Printf("ipstring: %s \n", fmt.Sprintf("%s:%d", s.ip, s.port))
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.ip, s.port))
	if err != nil {
		return err
	}

	s.sender, err = net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	return nil
}

func NewudpServer(ip string, port int, path string) Server {
	s := new(udpServer)
	s.ip = ip
	s.port = port
	s.file, _ = openOrCreateFile(path)
	return s
}

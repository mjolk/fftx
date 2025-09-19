package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

type Server interface {
	Recv(context.Context) (int, int, error)
	Start(context.Context) error
}

type udpServer struct {
	file   *os.File
	ip     string
	port   int
	sender net.PacketConn
}

func (s *udpServer) Recv(ctx context.Context) (int, int, error) {
	received := 0
	written := 0

	b := make([]byte, HEADER_SIZE)
	read, _, err := s.sender.ReadFrom(b[0:])
	if err != nil {
		return received, written, err
	}

	if read != 8 {
		return received, written, errors.New("Wrong header size")
	}

	size := binary.LittleEndian.Uint64(b)
	log.Printf("size: %d \n", size)

	tmp := make([]byte, 1024*32)
	received = 0
	receiving := 1
	receives := 0
	for receiving > 0 {

		recvd, _, err := s.sender.ReadFrom(tmp)
		received += recvd
		receives++
		if errors.Is(err, io.EOF) || received == int(size) {
			log.Printf("eof or done: %d \n", received)
			receiving = 0
		}
		if err != nil && receiving > 0 {
			return received, written, err
		}

		if recvd > 0 {
			w, err := s.file.Write(tmp[0:recvd])
			if err != nil {
				return received, w, err
			}
			written += w
			if w < recvd {
				log.Printf("writer cannot follow, dropped %d bytes \n", recvd-w)
			}
		}
		log.Printf("recieved %d written %d \n", received/1024, written/1024)
	}
	log.Printf("receives nr %d \n", receives)

	return received, written, nil
}

func (s *udpServer) Start(ctx context.Context) error {
	log.Printf("ipstring: %s \n", fmt.Sprintf("%s:%d", s.ip, s.port))
	var err error
	s.sender, err = listen(
		1024*4,
	).ListenPacket(ctx, "udp4", fmt.Sprintf("%s:%d", s.ip, s.port))
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
	log.Printf(
		"file %+v\n",
		s.file,
	)
	return s
}

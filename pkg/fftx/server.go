package fftx

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
	Start() error
}

type udpServer struct {
	file   *os.File
	ip     string
	port   int
	sender *net.UDPConn
}

func (s *udpServer) Recv(ctx context.Context) (int, int, error) {
	received := 0
	written := 0

	b := make([]byte, HEADER_SIZE)
	read, _, err := s.sender.ReadFromUDP(b[0:])
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
	for receiving > 0 {

		recvd, _, err := s.sender.ReadFromUDP(tmp)
		received += recvd
		if errors.Is(err, io.EOF) || received == int(size) {
			log.Printf("eof or done: %d \n", received)
			receiving = 0
		}
		if err != nil {
			return received, written, err
		}

		if recvd == 0 {
			continue
		}

		w, err := s.file.Write(tmp[0:recvd])
		if err != nil {
			return received, w, err
		}
		written += w
		if w < recvd {
			log.Printf("writer cannot follow, dropped %d bytes \n", recvd-w)
		}
	}

	return received, written, nil
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
	log.Printf(
		"file %+v\n",
		s.file,
	)
	return s
}

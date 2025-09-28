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
	"time"
)

type Server interface {
	Recv(context.Context, int64) (int, int, error)
	Start(context.Context) error
}

type chunk struct {
	ip     string
	path   string
	port   int
	offset int64
	id     int
}

type receiverChunk struct {
	chunk
	totalSize int64
	receiver  Server
}

func (c *receiverChunk) Start(ctx context.Context) error {
	c.receiver = NewudpServer(c.ip, c.port, c.path, c.totalSize)
	if err := c.receiver.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (c *receiverChunk) Recv(ctx context.Context) (int, int, error) {
	log.Printf("chunk %d receiving \n", c.id)
	return c.receiver.Recv(ctx, c.offset)
}

func NewReceiverChunk(
	ip, path string,
	port, id int,
	offset, size int64,
) *receiverChunk {
	chunk := new(receiverChunk)
	chunk.ip = ip
	chunk.path = path
	chunk.port = port
	chunk.offset = offset
	chunk.id = id
	chunk.totalSize = size
	return chunk
}

type udpServer struct {
	file      *os.File
	ip        string
	port      int
	sender    *net.UDPConn
	path      string
	totalSize int64
}

type completion struct {
	written int
	read    int
	err     error
	id      int
}

func (s *udpServer) ReadHeader() (int64, error) {
	b := make([]byte, HEADER_SIZE)
	read, _, err := s.sender.ReadFrom(b[0:])
	if err != nil {
		return int64(read), err
	}

	if read != 8 {
		return int64(read), errors.New("Wrong header size")
	}

	size := binary.LittleEndian.Uint64(b)
	log.Printf("size: %d \n", size)
	return int64(size), nil
}

func (s *udpServer) Recv(ctx context.Context, offset int64) (int, int, error) {
	received := 0
	written := 0
	tmp := make([]byte, 1024*32)
	received = 0
	receiving := 1
	receives := 0
	_, err := s.file.Seek(offset, 0)
	if err != nil {
		return 0, 0, err
	}
	for receiving > 0 {
		recvd, _, err := s.sender.ReadFrom(tmp)
		received += recvd
		receives++
		expected := min(CHAN_BUF_SIZE, s.totalSize-offset)
		/*log.Printf(
			"total size: %d, offset: %d, expected bytes: %d \n",
			s.totalSize,
			offset,
			expected,
		)*/
		if errors.Is(err, io.EOF) || received == int(expected) {
			log.Printf(
				"eof %t or done: %d \n",
				errors.Is(err, io.EOF),
				received,
			)
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
		// log.Printf("recieved %d written %d \n", received/1024, written/1024)
	}
	log.Printf("receives nr %d \n", receives)

	return received, written, nil
}

func (s *udpServer) Start(ctx context.Context) error {
	log.Printf("ipstring: %s \n", fmt.Sprintf("%s:%d", s.ip, s.port))
	s.file, _ = openOrCreateFile(s.path)
	log.Printf(
		"file %+v\n",
		s.file,
	)
	var err error
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", s.ip, s.port))
	s.sender, err = /*listen(
			1024*1024,
		)*/net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}
	s.sender.SetReadBuffer(1024 * 1024)

	return nil
}

func NewudpServer(ip string, port int, path string, size int64) *udpServer {
	s := new(udpServer)
	s.ip = ip
	s.port = port
	s.path = path
	s.totalSize = size
	return s
}

type receiver struct {
	host      string
	startPort int
	path      string
	size      int64
	client    *udpServer
}

func (r *receiver) Start(ctx context.Context) error {
	r.client = NewudpServer(r.host, r.startPort, r.path, r.size)
	if err := r.client.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (r *receiver) Recv(ctx context.Context) (int, int, error) {
	var err error
	r.size, err = r.client.ReadHeader()
	if err != nil {
		return 0, 0, err
	}
	log.Printf("receiving %d MB \n", r.size/(1024*1024))
	start := time.Now()
	var add int64
	rst := int64(r.size & (CHAN_BUF_SIZE - 1))
	if rst > 0 {
		add++
	}

	chunkCount := ((r.size - rst) / CHAN_BUF_SIZE) + add
	completions := make(chan completion)

	for c := range chunkCount {
		log.Printf("receiving on %s:%d \n", r.host, r.startPort+int(c+1))
		chunk := NewReceiverChunk(
			r.host,
			r.path,
			r.startPort+int(c+1),
			int(c),
			c*CHAN_BUF_SIZE,
			r.size,
		)

		go func(chnk *receiverChunk, compCh chan<- completion) {
			err := chnk.Start(ctx)
			if err != nil {
				compCh <- completion{err: err, id: chnk.id}
				return
			}
			written, read, err := chnk.Recv(ctx)
			if err != nil {
				compCh <- completion{err: err, id: chnk.id}
				return
			}
			log.Printf("chunk %d  \n", c)
			compCh <- completion{written: written, read: read, id: chnk.id}
		}(chunk, completions)
	}

	written := 0
	read := 0
	for range chunkCount {
		select {
		case complete := <-completions:
			if complete.err != nil {
				log.Printf("chunk error: %s \n", complete.err)
				return 0, 0, complete.err
			}
			log.Printf("chunk %d complete: %+v \n", complete.id, complete)
			read += complete.read
			written += complete.written
		case <-ctx.Done():
			return written, read, ctx.Err()
		}
	}

	end := time.Now()
	log.Printf("duration transfer: %f min\n", end.Sub(start).Minutes())

	return read, written, nil
}

func NewReceiver(path, host string, port int) *receiver {
	receiver := new(receiver)
	receiver.host = host
	receiver.startPort = port
	receiver.path = path
	return receiver
}

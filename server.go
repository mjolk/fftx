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
	receiver Server
}

func (c *receiverChunk) Start(ctx context.Context) error {
	c.receiver = NewudpServer(c.ip, c.port, c.path)
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
	offset int64,
) *receiverChunk {
	chunk := new(receiverChunk)
	chunk.ip = ip
	chunk.path = path
	chunk.port = port
	chunk.offset = offset
	chunk.id = id
	return chunk
}

type udpServer struct {
	file   *os.File
	ip     string
	port   int
	sender net.PacketConn
	path   string
}

type completion struct {
	written int
	read    int
	err     error
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
		if errors.Is(err, io.EOF) || received == int(CHAN_BUF_SIZE) {
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
	s.sender, err = listen(
		1024*4,
	).ListenPacket(ctx, "udp4", fmt.Sprintf("%s:%d", s.ip, s.port))
	if err != nil {
		return err
	}

	return nil
}

func NewudpServer(ip string, port int, path string) *udpServer {
	s := new(udpServer)
	s.ip = ip
	s.port = port
	s.path = path
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
	r.client = NewudpServer(r.host, r.startPort, r.path)
	if err := r.client.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (r *receiver) Recv(ctx context.Context) (int, int, error) {
	size, err := r.client.ReadHeader()
	if err != nil {
		return 0, 0, err
	}
	log.Printf("receiving %d MB \n", size/(1024*1024))

	var add int64
	rst := int64(size & (CHAN_BUF_SIZE - 1))
	if rst > 0 {
		add++
	}

	chunkCount := ((size - rst) / CHAN_BUF_SIZE)
	completions := make(chan completion)

	for c := range chunkCount {
		log.Printf("receiving on %s:%d \n", r.host, r.startPort+int(c+1))
		chunk := NewReceiverChunk(
			r.host,
			r.path,
			r.startPort+int(c+1),
			int(c),
			c*CHAN_BUF_SIZE,
		)

		go func(chnk *receiverChunk, compCh chan<- completion) {
			err := chnk.Start(ctx)
			if err != nil {
				compCh <- completion{err: err}
				return
			}
			written, read, err := chnk.Recv(ctx)
			if err != nil {
				compCh <- completion{err: err}
				return
			}
			compCh <- completion{written: written, read: read}
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
			log.Printf("chunk complete: %+v \n", complete)
			read += complete.read
			written += complete.written
		case <-ctx.Done():
			return written, read, ctx.Err()
		}
	}

	return read, written, nil
}

func NewReceiver(path, host string, port int) *receiver {
	receiver := new(receiver)
	receiver.host = host
	receiver.startPort = port
	receiver.path = path
	return receiver
}

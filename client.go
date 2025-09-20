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

const (
	HEADER_SIZE = 8
	// this depends on your network, lower number = higher flow -> more chance packets get bounced.
	// A fluctuating flow would be faster but then we'd have to real time measure or detect packet loss and manage congestion
	// which is a bunch of extra code/time, might as well use tcp then
	// FLOW = 1 * time.Millisecond
)

const (
	CHAN_BUF_SIZE = int64(1024 * 1024 * 1024)
	BLOCK_SIZE    = int64(1024 * 16)
)

type Sender interface {
	Send(context.Context, int64) (int, error)
	Start(context.Context) error
}

type senderChunk struct {
	chunk
	sender Sender
}

func (c *senderChunk) Start(ctx context.Context) (err error) {
	c.sender, err = NewClient(c.path, c.ip, c.port)
	if err := c.sender.Start(ctx); err != nil {
		return err
	}
	return
}

func (c *senderChunk) Send(ctx context.Context) (int, error) {
	log.Printf("chunk %d sending \n", c.id)
	return c.sender.Send(ctx, c.offset)
}

func NewSenderChunk(ip, path string, port int, offset int64) *senderChunk {
	chunk := new(senderChunk)
	chunk.ip = ip
	chunk.path = path
	chunk.port = port
	chunk.offset = offset
	return chunk
}

type client struct {
	path string
	ip   string
	port int
	file *os.File
	sink *net.UDPConn
	sent int64
	addr *net.UDPAddr
}

func (c *client) Send(ctx context.Context, offset int64) (int, error) {
	tmp := make([]byte, BLOCK_SIZE)
	written := 0
	// ticker := time.NewTicker(FLOW)
	done := false
	rst := CHAN_BUF_SIZE & (BLOCK_SIZE - 1)
	var xtra int64
	if rst > 0 {
		xtra++
	}
	nrblocks := (CHAN_BUF_SIZE-rst)/BLOCK_SIZE + xtra
	log.Printf("rest: %d xtra: %d nrblocks: %d \n", rst, xtra, nrblocks)
	// sf := io.NewSectionReader(c.file, 1024*1024, info.Size())
	sends := 0
	_, err := c.file.Seek(offset, 0)
	if err != nil {
		return 0, err
	}
	for range nrblocks {
		read, err := c.file.Read(tmp)
		if errors.Is(err, io.EOF) {
			done = true
		}
		if err != nil && !done {
			log.Printf("not really done ---------------->>> ")
			return written, err
		}

		if read > 0 {
			w, err := c.sink.Write(tmp[0:read])
			if w < 0 || read < w {
				w = 0
				if err == nil {
					return written, errors.New("Invalid write")
				}
			}
			written += w
			if err != nil {
				return written, err
			}
			sends++
		}
		if done {
			return written, nil
		}
		// log.Printf("WRITTEN %d \n", written/1024)
		time.Sleep(1 * time.Millisecond)

	}

	// log.Printf("nr sends: %d ---------------->>> \n", sends)
	return written, nil
}

func (c *client) Start(ctx context.Context) error {
	var err error
	c.file, err = openReadFile(c.path)
	if err != nil {
		return err
	}
	conn, err := /*sender(
			1024,
		)*/net.Dial("udp4", fmt.Sprintf("%s:%d", c.ip, c.port))
	if err != nil {
		return err
	}
	udp, ok := conn.(*net.UDPConn)
	if !ok {
		return errors.New("no udp conn")
	}
	c.sink = udp
	log.Printf(
		"------------>>>udp sink %+v \n",
		c.sink,
	)
	return nil
}

func NewClient(
	path string,
	targetHost string,
	port int,
) (c *client, err error) {
	c = new(client)
	c.path = path
	c.ip = targetHost
	c.port = port
	return
}

type send struct {
	chunkCount int64
	host       string
	startPort  int
	path       string
	size       int64
	client     *client
}

func (s *send) Start(ctx context.Context) error {
	var err error
	s.client, err = NewClient(s.path, s.host, s.startPort)
	if err != nil {
		return err
	}

	if err := s.client.Start(ctx); err != nil {
		return err
	}

	info, err := s.client.file.Stat()
	if err != nil {
		return err
	}

	s.size = info.Size()
	rst := s.size & (CHAN_BUF_SIZE - 1)
	var add int64
	if rst < 0 {
		add++
	}

	s.chunkCount = (s.size-rst)/CHAN_BUF_SIZE + add
	b := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint64(b, uint64(s.size))
	written, err := s.client.sink.Write(b)
	if err != nil {
		return err
	}
	log.Printf("written header of size: %d  file size: %d\n", written, s.size)
	return nil
}

func (s *send) Send(ctx context.Context) (int, error) {
	completions := make(chan completion)
	for c := range s.chunkCount {
		log.Printf("sending to %s:%d \n", s.host, s.startPort+int(c+1))
		chunk := NewSenderChunk(
			s.host,
			s.path,
			s.startPort+int(c+1),
			c*CHAN_BUF_SIZE,
		)

		log.Printf("offset %d \n", c*CHAN_BUF_SIZE)

		go func(chnk *senderChunk, cmp chan<- completion) {
			err := chnk.Start(ctx)
			if err != nil {
				cmp <- completion{err: err}
				return
			}

			written, err := chnk.Send(ctx)
			if err != nil {
				cmp <- completion{err: err}
				return
			}

			cmp <- completion{written: written}
		}(chunk, completions)
	}

	written := 0
	for range s.chunkCount {
		select {
		case comp := <-completions:
			if comp.err != nil {
				log.Printf("chunk error: %s \n", comp.err)
				return 0, comp.err
			}
			log.Printf("chunk complete: %+v \n", comp)
			written += comp.written
		case <-ctx.Done():
			return written, ctx.Err()
		}
	}

	return written, nil
}

func NewSend(path, host string, port int) *send {
	send := new(send)
	send.path = path
	send.host = host
	send.startPort = port
	return send
}

package fftx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const HEADER_SIZE = 8

type client struct {
	path string
	ip   string
	port int
	file *os.File
	sink *net.UDPConn
	sent int64
	addr *net.UDPAddr
}

func (c *client) Send() (int, error) {
	written := 0
	info, err := c.file.Stat()
	if err != nil {
		return written, err
	}

	b := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint64(b, uint64(info.Size()))
	written, err = c.sink.Write(b)
	if err != nil {
		return written, err
	}
	log.Printf("written header of size: %d \n", written)

	tmp := make([]byte, 1024*32)
	written = 0
	ticker := time.NewTicker(1 * time.Millisecond)
	// sf := io.NewSectionReader(c.file, 1024*1024, info.Size())
	for range ticker.C {
		read, err := c.file.Read(tmp)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return written, err
		}

		if read == 0 {
			continue
		}

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
	}

	return written, nil
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
	c.file, err = openReadFile(path)
	if err != nil {
		return nil, err
	}
	c.addr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.ip, c.port))
	if err != nil {
		return nil, err
	}
	c.sink, err = net.DialUDP("udp", nil, c.addr)
	log.Printf(
		"------------>>>udp sink %+v \n",
		c.sink,
	)
	if err != nil {
		return nil, err
	}
	return
}

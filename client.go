package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	HEADER_SIZE = 8
	// this depends on your network, lower number = higher flow -> more chance packets get bounced.
	// A fluctuating flow would be faster but then we'd have to real time measure or detect packet loss and manage congestion
	// which is a bunch of extra code/time, might as well use tcp then
	// FLOW = 1 * time.Millisecond
)

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

	size := info.Size()
	b := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint64(b, uint64(size))
	written, err = c.sink.Write(b)
	if err != nil {
		return written, err
	}
	log.Printf("written header of size: %d \n", written)

	tmp := make([]byte, 1024*32)
	written = 0
	// ticker := time.NewTicker(FLOW)
	done := false
	block := int64(1024 * 1024 * 256)
	rst := int64(size & (block - 1))
	var xtra int64
	if rst > 0 {
		xtra++
	}
	nrblocks := (size-rst)/block + xtra
	log.Printf("rest: %d xtra: %d nrblocks: %d \n", rst, xtra, nrblocks)
	var cur int64
	// sf := io.NewSectionReader(c.file, 1024*1024, info.Size())
	sends := 0
	for b := range nrblocks {
		cur = 0
		log.Printf("BLOCK %d \n", b)
		for cur <= block {
			read, err := c.file.Read(tmp)
			if errors.Is(err, io.EOF) {
				done = true
			}
			if err != nil && !done {
				log.Printf("not really done ---------------->>> ")
				return written, err
			}

			if read > 0 {
				cur += int64(read)
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
		}
		log.Printf("WRITTEN %d \n", written)
		// time.Sleep(600 * time.Millisecond)

	}

	log.Printf("nr sends: %d ---------------->>> \n", sends)
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

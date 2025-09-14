package fftx

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

type client struct {
	path string
	ip   string
	port int
	file *os.File
	sink *net.UDPConn
	sent int64
}

func (c *client) Send() error {
	/*defer func() {
		c.sink.Close()
		c.file.Close()
	}()*/

	w, err := io.Copy(c.sink, c.file)
	if err != nil {
		log.Printf("error copying data %s \n", err)
		return err
	}
	c.sent += w
	log.Printf("sent %d mb\n", w/1024)
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
	c.file, err = openReadFile(path)
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.ip, c.port))
	if err != nil {
		return nil, err
	}
	c.sink, err = net.DialUDP("udp", nil, addr)
	log.Printf(
		"------------>>>udp sink %+v \n",
		c.sink,
	)
	if err != nil {
		return nil, err
	}
	return
}

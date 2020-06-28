package main

import (
	"bufio"
	"log"
	"net"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "localhost:55555")
	if err != nil {
		log.Fatal("Listen:", err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal("Accept: ", err)
		}

		go procesTCPConn(c)
	}
}

func procesTCPConn(c net.Conn) {
	defer c.Close()
	log.Println("remote addr = ", c.RemoteAddr())

	c.Write([]byte("220 Yves FTP Ready\r\n"))

	ftpConn := &ftpConn{c}
	ftpConn.processFTPConn()
}

type ftpConn struct {
	net.Conn
}

func (c *ftpConn) processFTPConn() {
	s := bufio.NewScanner(c)
	for s.Scan() {
		line := s.Text()
		log.Printf("line=%q", line)
		splits := strings.SplitN(line, " ", 2)
		cmd := splits[0]
		args := splits[1]
		log.Printf("cmd=%q, args=%q", cmd, args)
	}
}

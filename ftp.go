package main

import (
	"bufio"
	"fmt"
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

		tcpc := c.(*net.TCPConn)
		go procesTCPConn(tcpc)
	}
}

func procesTCPConn(c *net.TCPConn) {
	defer c.Close()
	log.Println("remote addr = ", c.RemoteAddr())

	c.Write([]byte("220 Yves FTP Ready\r\n"))

	ftpConn := &ftpConn{c}
	ftpConn.processFTPConn()
}

type ftpConn struct {
	*net.TCPConn
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

		switch cmd {
		case "USER":
			c.procUserCmd(args)
		}
	}
}

func (c *ftpConn) procUserCmd(args string) {
	log.Printf("procUserCmd(%q) %#v", args, *c)
	fmt.Fprintf(c, "230 User %s logged in, proceed.", args)
}

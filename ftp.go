package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
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

	curWorkDir, err := os.Getwd()
	if err != nil {
		log.Fatal("os.Getwd:", err)
	}
	ftpConn := &ftpConn{c, curWorkDir}
	ftpConn.processFTPConn()
}

type ftpConn struct {
	*net.TCPConn
	curWorkDir string
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
		case "CWD":
			c.procCWDCmd(args)
		}
	}
}

func (c *ftpConn) procUserCmd(args string) {
	log.Printf("procUserCmd(%q) %#v", args, *c)
	fmt.Fprintf(c, "230 User %s logged in, proceed.\r\n", args)
}

func (c *ftpConn) procCWDCmd(args string) {
	log.Printf("procCWDCmd(%q) %#v", args, *c)
	if len(args) == 0 {
		fmt.Fprintf(c, "501 Empty parameters.\r\n")
	}
	newCurWorkDir := c.curWorkDir + "/" + args
	_, err := os.Stat(newCurWorkDir)
	if err != nil {
		log.Print(err)
		fmt.Fprintf(c, "550 new working dir does not exist.\r\n")
	}
	c.curWorkDir = newCurWorkDir
	fmt.Fprintf(c, "200 Current workdir changed.\r\n")
}

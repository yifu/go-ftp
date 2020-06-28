package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
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
		log.Printf(">> %q", line)
		splits := strings.SplitN(line, " ", 2)
		cmd := splits[0]
		var args string
		if len(splits) > 1 {
			args = splits[1]
		}
		log.Printf("cmd=%q, args=%q", cmd, args)

		switch cmd {
		case "USER":
			c.procUserCmd(args)
		case "CWD":
			c.procCWDCmd(args)
		case "QUIT":
			c.procQuitCmd(args)
			return
		}
	}
}

func (c *ftpConn) procUserCmd(args string) {
	log.Printf("procUserCmd(%q) %#v", args, *c)
	c.reply(fmt.Sprintf("230 User %s logged in, proceed.", args))
}

func (c *ftpConn) procCWDCmd(args string) {
	log.Printf("procCWDCmd(%q) %#v", args, *c)
	if len(args) == 0 {
		c.reply("501 Empty parameters.")
		return
	}
	newCurWorkDir := filepath.Clean(c.curWorkDir + "/" + args)
	_, err := os.Stat(newCurWorkDir)
	if err != nil {
		log.Print("os.Stat: ", err)
		c.reply("550 new working dir does not exist.")
		return
	}
	c.curWorkDir = newCurWorkDir
	c.reply("200 Current workdir changed.")
}

func (c *ftpConn) procQuitCmd(args string) {
	c.reply("221 Bye.")
}

func (c *ftpConn) reply(line string) {
	line += "\r\n"
	log.Printf("<< %q", line)
	fmt.Fprint(c, line)
}

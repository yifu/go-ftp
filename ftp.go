package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
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
	ftpConn := &ftpConn{c, curWorkDir, ""}
	ftpConn.processFTPConn()
}

type ftpConn struct {
	*net.TCPConn
	curWorkDir string
	dataAddr   string
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
		case "PORT":
			c.procPortCmd(args)
		case "LIST":
			c.procListCmd(args)
		case "RETR":
			c.procRetrCmd(args)
		default:
			c.procUnknownCmd(cmd, args)
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
	newCurWorkDir := filepath.Clean(filepath.Join(c.curWorkDir, args))
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

func (c *ftpConn) procPortCmd(args string) {
	arguments := strings.Split(args, ",")
	if len(arguments) != 6 {
		c.reply(fmt.Sprintf("501 %v is the wrong number of arguments for the port cmd.", len(arguments)))
	}

	portMSB, err := strconv.Atoi(arguments[4])
	if err != nil {
		log.Print(arguments[4], " port MSB is not a number.")
		c.reply("501 Syntax error in parameters or arguments")
		return
	}
	portLSB, err := strconv.Atoi(arguments[5])
	if err != nil {
		log.Print(arguments[5], " port LSB is not a number.")
		c.reply("501 Syntax error in parameters or arguments")
		return
	}
	port := portMSB
	port <<= 8
	port += portLSB

	//c.dataAddr = fmt.Sprintf("%v.%v.%v.%v:%v", arguments[0:4]..., port)
	c.dataAddr = fmt.Sprintf("%v.%v.%v.%v:%v", arguments[0], arguments[1], arguments[2], arguments[3], port)
	log.Print("data addr is ", c.dataAddr)
	c.reply("200 Cmd ok")
}

func (c *ftpConn) procListCmd(args string) {
	targetPath := filepath.Clean(filepath.Join(c.curWorkDir, args))
	log.Printf("Target path %q", targetPath)

	files, err := ioutil.ReadDir(targetPath)
	if err != nil {
		log.Printf("ioutil.ReadDir(%q): %v", targetPath, err)
		c.reply("550 " + err.Error())
		return
	}
	c.reply("150 Opening data connection")

	dataConn, err := c.connectToDataChan()
	if err != nil {
		c.reply("425 Cannot open data connection.")
		return
	}
	defer dataConn.Close()

	for _, file := range files {
		fmt.Fprintf(dataConn, "%v%v", file.Name(), c.EOL())
	}
	fmt.Fprintf(dataConn, "%v", c.EOL())

	c.reply("226 Closing data connection")
}

func (c *ftpConn) procRetrCmd(args string) {
	targetPath := filepath.Clean(filepath.Join(c.curWorkDir, args))
	log.Printf("Target path %q", targetPath)

	f, err := os.Open(targetPath)
	if err != nil {
		log.Printf("os.Open(%q): %v", targetPath, err)
		c.reply("550 Bad file")
		return
	}

	c.reply("150 Open data connection")
	dataConn, err := c.connectToDataChan()
	if err != nil {
		log.Printf("connectToDataChan(): %v", err)
		c.reply("425 Can't open data connection")
		return
	}
	defer dataConn.Close()

	if _, err := io.Copy(dataConn, f); err != nil {
		log.Printf("io.Copy(): %v", err)
		c.reply("426 Abort copy")
		return
	}

	c.reply("226 Close data connection")
}

func (c *ftpConn) connectToDataChan() (net.Conn, error) {
	return net.Dial("tcp", c.dataAddr)
}

func (c *ftpConn) EOL() string {
	return "\r\n"
}

func (c *ftpConn) procUnknownCmd(cmd, args string) {
	log.Printf("Unknown cmd %v args:[%v]", cmd, args)
	c.reply("502 Not implemented")
}

func (c *ftpConn) reply(line string) {
	line += c.EOL()
	log.Printf("<< %q", line)
	fmt.Fprint(c, line)
}

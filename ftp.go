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

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("os.Getwd:", err)
	}
	ftpConn := &ftpConn{c, wd, wd, "", false}
	ftpConn.processFTPConn()
}

type ftpConn struct {
	*net.TCPConn
	jailDir, curWorkDir string
	dataAddr            string
	isDataTypeBinary    bool
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
		case "TYPE":
			c.procTypeCmd(args)
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

	targetPath := c.computeTargetPath(args)
	if !strings.HasPrefix(targetPath, c.jailDir) {
		c.reply("550 new working dir is out of jail.")
	}

	fi, err := os.Stat(targetPath)
	if err != nil {
		log.Print("os.Stat: ", err)
		c.reply("550 new working dir does not exist.")
		return
	}
	if !fi.IsDir() {
		c.reply("550 arg is not a directory.")
		return
	}

	c.curWorkDir = targetPath
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
	targetPath := c.computeTargetPath(args)
	if !strings.HasPrefix(targetPath, c.jailDir) {
		c.reply("550 new working dir is out of jail.")
	}

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
	if len(args) == 0 {
		c.reply("501 Empty parameters.")
		return
	}

	targetPath := c.computeTargetPath(args)
	if !strings.HasPrefix(targetPath, c.jailDir) {
		c.reply("550 new working dir is out of jail.")
	}

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

	if c.isDataTypeBinary {
		log.Printf("Copying with io.Copy()")
		if _, err := io.Copy(dataConn, f); err != nil {
			log.Printf("io.Copy(): %v", err)
			c.reply("426 Abort copy")
			return
		}
	} else {
		log.Printf("Copying with scanner")
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fmt.Fprintf(dataConn, "%s\r\n", scanner.Text())
		}
	}

	c.reply("226 Close data connection")
}

func (c *ftpConn) procTypeCmd(args string) {
	cmdArguments := strings.Split(args, " ")
	typ := cmdArguments[0]

	if len(typ) != 1 {
		c.reply("501 Syntax error in parameters or arguments.")
		return
	}

	switch rune(typ[0]) {
	case 'A':
		formcodes := cmdArguments[1:]
		if len(formcodes) > 1 {
			c.reply("500 Syntax error, command unrecognized.")
			return
		} else if len(formcodes) == 1 {
			c.reply("504 Command not implemented for that parameter.")
			return
		}
		c.isDataTypeBinary = false
	case 'I':
		imgArguments := cmdArguments[1:]
		if len(imgArguments) > 0 {
			c.reply("500 Syntax error, command unrecognized.")
			return
		}
		c.isDataTypeBinary = true
	case 'L':
		localArguments := cmdArguments[1:]
		if len(localArguments) > 1 {
			c.reply("500 Syntax error, command unrecognized.")
			return
		}
		n, err := strconv.Atoi(localArguments[0])
		if err != nil {
			c.reply("500 Syntax error, command unrecognized.")
			return
		}

		if n != 8 {
			c.reply("504 Command not implemented for that parameter.")
			return
		}
		c.isDataTypeBinary = true

	case 'E':
		c.reply("504 Command not implemented for that parameter.")
		return
	}

	log.Printf("c=%+v", *c)
	c.reply("200 Cmd ok")
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

func (c *ftpConn) computeTargetPath(arg string) string {
	arg = strings.Trim(arg, "")

	if filepath.IsAbs(arg) {
		// args is an absolute path.
		log.Printf("%q is an absolute path.", arg)
		return filepath.Clean(filepath.Join(c.jailDir, arg))
	}
	// args is a relative path.
	log.Printf("%q is a relative path.", arg)
	return filepath.Clean(filepath.Join(c.curWorkDir, arg))
}

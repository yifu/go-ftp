package main

import (
	"log"
	"net"
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
		log.Println("remote addr = ", c.RemoteAddr())

		c.Write([]byte("220 Yves FTP Ready\r\n"))

		b := make([]byte, 40)
		n, err := c.Read(b)
		if err != nil {
			log.Fatal("Read: ", err)
		}

		log.Printf("b:=%q[%v]", string(b[:n-2]), n)
	}
}

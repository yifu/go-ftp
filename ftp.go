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
	}
}

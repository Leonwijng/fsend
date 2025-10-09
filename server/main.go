package main

import (
	"fmt"
	"log"
	"net"
)

const (
	putfile uint8 = iota
	listFiles
	streamFile
)

func main() {
	lis, err := net.Listen("tcp", ":3001")
	if err != nil {
		log.Fatalln(err)
	}
	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}


	}
}

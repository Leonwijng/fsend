package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

const (
	putfile uint8 = iota
	listFiles
	streamFile
	ping
	bye
)

type ServerContext struct {
	clients map[net.Conn]struct{}
	lis     net.Listener
	mu      sync.Mutex
}

func putFile(conn net.Conn) {
	var (
		fnameSize uint8
		fname     string

		fsize   uint64
		bufSize uint32
	)

	err := binary.Read(conn, binary.LittleEndian, &fnameSize)
	if err != nil {
		return
	}

	var fnameBytes = make([]byte, fnameSize)
	_, err = conn.Read(fnameBytes)
	if err != nil {
		return
	}
	fname = string(fnameBytes)

	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()

	err = binary.Read(conn, binary.LittleEndian, &fsize)
	if err != nil {
		return
	}

	err = binary.Read(conn, binary.LittleEndian, &bufSize)
	if err != nil {
		return
	}

	if bufSize == 0 {
		bufSize = 32 * 1024
	}

	buf := make([]byte, int(bufSize))
	lr := io.LimitReader(conn, int64(fsize))
	_, err = io.CopyBuffer(f, lr, buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return
		}

		fmt.Println("copy error:", err)
		return
	}
}

func (s *ServerContext) handleClient(conn net.Conn) {
	s.mu.Lock()
	s.clients[conn] = struct{}{}
	s.mu.Unlock()

	for {
		var o uint8
		err := binary.Read(conn, binary.LittleEndian, &o)
		if err != nil {
			fmt.Println(err)
			return
		}

		switch o {
		case putfile:
			putFile(conn)
		case listFiles:
		case streamFile:
		case ping:
			_, err = conn.Write([]byte("pong"))
			if err != nil {
				fmt.Println(err)
				return
			}
		case bye:
			conn.Close()
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
		}
	}

}

func (s *ServerContext) Listen(address string) (err error) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go s.handleClient(conn)
	}
}

func main() {
	ctx := ServerContext{
		clients: make(map[net.Conn]struct{}),
	}

	fmt.Println("listening on :3002 ")
	panic(ctx.Listen(":3002"))
}

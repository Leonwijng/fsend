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
	register // New: register client UUID
)

type ClientInfo struct {
	uuid string
	conn net.Conn
}

type ServerContext struct {
	clients map[net.Conn]*ClientInfo // Changed to store client info
	lis     net.Listener
	mu      sync.Mutex
}

func putFile(conn net.Conn, targetUUID string) {
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

	// Save file in UUID directory
	filePath := getUUIDDirectory(targetUUID) + "/" + fname
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
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

	fmt.Printf("✓ Saved file %s for UUID %s\n", fname, targetUUID)
}

func (s *ServerContext) handleClient(conn net.Conn) {
	s.mu.Lock()
	s.clients[conn] = &ClientInfo{conn: conn}
	s.mu.Unlock()

	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
	}()

	var clientUUID string

	for {
		var o uint8
		err := binary.Read(conn, binary.LittleEndian, &o)
		if err != nil {
			fmt.Println(err)
			return
		}

		switch o {
		case register:
			// Read UUID length
			var uuidLen uint8
			err = binary.Read(conn, binary.LittleEndian, &uuidLen)
			if err != nil {
				fmt.Println("Error reading UUID length:", err)
				return
			}

			// Read UUID
			uuidBytes := make([]byte, uuidLen)
			_, err = conn.Read(uuidBytes)
			if err != nil {
				fmt.Println("Error reading UUID:", err)
				return
			}

			clientUUID = string(uuidBytes)
			s.mu.Lock()
			s.clients[conn].uuid = clientUUID
			s.mu.Unlock()

			// Ensure directory exists for this UUID
			err = ensureUUIDDirectory(clientUUID)
			if err != nil {
				fmt.Println("Error creating UUID directory:", err)
				return
			}

			fmt.Printf("✓ Client registered: %s\n", clientUUID)

		case putfile:
			if clientUUID == "" {
				fmt.Println("Error: Client not registered")
				return
			}
			putFile(conn, clientUUID)

		case listFiles:
			if clientUUID == "" {
				fmt.Println("Error: Client not registered")
				return
			}
			err = handleListFiles(conn, clientUUID)
			if err != nil {
				fmt.Println(err)
				return
			}

		case streamFile:

		case ping:
			_, err = conn.Write([]byte("pong"))
			if err != nil {
				fmt.Println(err)
				return
			}

		case bye:
			return
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
	// Ensure files directory exists
	err := ensureFilesDirectory()
	if err != nil {
		panic(err)
	}

	ctx := ServerContext{
		clients: make(map[net.Conn]*ClientInfo),
	}

	fmt.Println("Listening on :3002")
	panic(ctx.Listen(":3002"))
}

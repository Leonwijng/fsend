package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
)

const (
	putfile uint8 = iota
	listFiles
	streamFile
	ping
	bye
	register   // Register client UUID
	sendToUUID // Send file to another client's UUID
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

	// Save file in UUID directory (cross-platform path)
	filePath := filepath.Join(getUUIDDirectory(targetUUID), fname)
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

// handleSendToUUID receives a file from one client and saves it to another client's UUID directory
func (s *ServerContext) handleSendToUUID(conn net.Conn, senderUUID string) error {
	// Read target UUID length
	var targetUUIDLen uint8
	err := binary.Read(conn, binary.LittleEndian, &targetUUIDLen)
	if err != nil {
		return fmt.Errorf("failed to read target UUID length: %w", err)
	}

	// Read target UUID
	targetUUIDBytes := make([]byte, targetUUIDLen)
	_, err = conn.Read(targetUUIDBytes)
	if err != nil {
		return fmt.Errorf("failed to read target UUID: %w", err)
	}
	targetUUID := string(targetUUIDBytes)

	// Read filename length
	var fnameLen uint8
	err = binary.Read(conn, binary.LittleEndian, &fnameLen)
	if err != nil {
		return fmt.Errorf("failed to read filename length: %w", err)
	}

	// Read filename
	fnameBytes := make([]byte, fnameLen)
	_, err = conn.Read(fnameBytes)
	if err != nil {
		return fmt.Errorf("failed to read filename: %w", err)
	}
	fname := string(fnameBytes)

	// Read file size
	var fsize uint64
	err = binary.Read(conn, binary.LittleEndian, &fsize)
	if err != nil {
		return fmt.Errorf("failed to read file size: %w", err)
	}

	// Ensure target UUID directory exists
	err = ensureUUIDDirectory(targetUUID)
	if err != nil {
		return fmt.Errorf("failed to create target UUID directory: %w", err)
	}

	// Create file in target UUID's directory
	filePath := filepath.Join(getUUIDDirectory(targetUUID), fname)
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Receive file data
	buf := make([]byte, 32*1024)
	remaining := fsize
	for remaining > 0 {
		toRead := len(buf)
		if uint64(toRead) > remaining {
			toRead = int(remaining)
		}

		n, err := conn.Read(buf[:toRead])
		if err != nil {
			return fmt.Errorf("failed to read file data: %w", err)
		}

		_, err = f.Write(buf[:n])
		if err != nil {
			return fmt.Errorf("failed to write file data: %w", err)
		}

		remaining -= uint64(n)
	}

	fmt.Printf("✓ File %s sent from %s to %s (%d bytes)\n", fname, senderUUID, targetUUID, fsize)
	return nil
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
			if clientUUID == "" {
				fmt.Println("Error: Client not registered")
				return
			}
			err = handleStreamFile(conn, clientUUID)
			if err != nil {
				fmt.Println(err)
				return
			}

		case sendToUUID:
			if clientUUID == "" {
				fmt.Println("Error: Client not registered")
				return
			}
			err = s.handleSendToUUID(conn, clientUUID)
			if err != nil {
				fmt.Println(err)
				return
			}

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

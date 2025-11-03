package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
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

const uidFile = ".fsend_uid"

// Client represents a connection to the fsend server
type Client struct {
	conn    net.Conn
	address string
	uid     string // Client UUID
}

// loadOrCreateUID loads the UID from file or creates a new one
func loadOrCreateUID() (string, error) {
	// Try to read existing UID
	data, err := os.ReadFile(uidFile)
	if err == nil {
		uid := string(data)
		fmt.Println("✓ Using existing client UID:", uid)
		return uid, nil
	}

	// UID doesn't exist, create new one
	if os.IsNotExist(err) {
		uid := uuid.New().String()

		// Save UID to file (read-only)
		err = os.WriteFile(uidFile, []byte(uid), 0444)
		if err != nil {
			return "", fmt.Errorf("failed to save UID: %w", err)
		}

		fmt.Println("✓ Generated new client UID:", uid)
		return uid, nil
	}

	return "", fmt.Errorf("failed to read UID file: %w", err)
}

// NewClient creates a new client instance
func NewClient(address string) (*Client, error) {
	uid, err := loadOrCreateUID()
	if err != nil {
		return nil, err
	}

	return &Client{
		address: address,
		uid:     uid,
	}, nil
}

// Connect establishes a connection to the server
func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.conn = conn

	// Register UUID with server
	err = c.registerUID()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to register with server: %w", err)
	}

	return nil
}

// registerUID sends the client's UUID to the server
func (c *Client) registerUID() error {
	// Send register command
	err := binary.Write(c.conn, binary.LittleEndian, register)
	if err != nil {
		return err
	}

	// Send UUID length
	uuidLen := uint8(len(c.uid))
	err = binary.Write(c.conn, binary.LittleEndian, uuidLen)
	if err != nil {
		return err
	}

	// Send UUID
	_, err = c.conn.Write([]byte(c.uid))
	if err != nil {
		return err
	}

	return nil
}

// Ping sends a ping to verify the connection
func (c *Client) Ping() error {
	if c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	err := binary.Write(c.conn, binary.LittleEndian, ping)
	if err != nil {
		return err
	}

	// Read pong response
	buf := make([]byte, 4)
	_, err = c.conn.Read(buf)
	if err != nil {
		return err
	}

	if string(buf) != "pong" {
		return fmt.Errorf("unexpected response: %s", string(buf))
	}

	return nil
}

// Close closes the connection to the server
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	// Send bye message
	binary.Write(c.conn, binary.LittleEndian, bye)
	time.Sleep(100 * time.Millisecond)

	return c.conn.Close()
}

// GetConnection returns the underlying connection for file operations
func (c *Client) GetConnection() net.Conn {
	return c.conn
}

// ListFiles requests and returns a list of available files from the server
func (c *Client) ListFiles() ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to server")
	}

	// Send listFiles command
	err := binary.Write(c.conn, binary.LittleEndian, listFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to send listFiles command: %w", err)
	}

	// Read number of files
	var fileCount uint32
	err = binary.Read(c.conn, binary.LittleEndian, &fileCount)
	if err != nil {
		return nil, fmt.Errorf("failed to read file count: %w", err)
	}

	// Read each filename
	files := make([]string, 0, fileCount)
	for i := uint32(0); i < fileCount; i++ {
		var nameLen uint8
		err = binary.Read(c.conn, binary.LittleEndian, &nameLen)
		if err != nil {
			return nil, fmt.Errorf("failed to read filename length: %w", err)
		}

		nameBuf := make([]byte, nameLen)
		_, err = c.conn.Read(nameBuf)
		if err != nil {
			return nil, fmt.Errorf("failed to read filename: %w", err)
		}

		files = append(files, string(nameBuf))
	}

	return files, nil
}

// GetUID returns the client's UID
func (c *Client) GetUID() string {
	return c.uid
}

// DownloadFile downloads a specific file from the server
func (c *Client) DownloadFile(filename string, savePath string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	// Send streamFile command
	err := binary.Write(c.conn, binary.LittleEndian, streamFile)
	if err != nil {
		return fmt.Errorf("failed to send streamFile command: %w", err)
	}

	// Send filename length
	fnameLen := uint8(len(filename))
	err = binary.Write(c.conn, binary.LittleEndian, fnameLen)
	if err != nil {
		return fmt.Errorf("failed to send filename length: %w", err)
	}

	// Send filename
	_, err = c.conn.Write([]byte(filename))
	if err != nil {
		return fmt.Errorf("failed to send filename: %w", err)
	}

	// Read file size
	var fsize uint64
	err = binary.Read(c.conn, binary.LittleEndian, &fsize)
	if err != nil {
		return fmt.Errorf("failed to read file size: %w", err)
	}

	// Check if file was found (size 0 = not found)
	if fsize == 0 {
		return fmt.Errorf("file not found on server: %s", filename)
	}

	// Create local file
	f, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer f.Close()

	// Read file data
	buf := make([]byte, 32*1024)
	remaining := fsize
	for remaining > 0 {
		toRead := len(buf)
		if uint64(toRead) > remaining {
			toRead = int(remaining)
		}

		n, err := c.conn.Read(buf[:toRead])
		if err != nil {
			return fmt.Errorf("failed to read file data: %w", err)
		}

		_, err = f.Write(buf[:n])
		if err != nil {
			return fmt.Errorf("failed to write file data: %w", err)
		}

		remaining -= uint64(n)
	}

	fmt.Printf("✓ Downloaded %s (%d bytes)\n", filename, fsize)
	return nil
}

// SendFileToUUID sends a file to another client's UUID
func (c *Client) SendFileToUUID(filePath string, targetUUID string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("cannot send directory: %s", filePath)
	}

	filename := fileInfo.Name()
	filesize := fileInfo.Size()

	// Validate lengths
	if len(filename) > 255 {
		return fmt.Errorf("filename too long (max 255 chars)")
	}
	if len(targetUUID) > 255 {
		return fmt.Errorf("UUID too long")
	}

	// Send sendToUUID command
	err = binary.Write(c.conn, binary.LittleEndian, sendToUUID)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Send target UUID length
	targetUUIDLen := uint8(len(targetUUID))
	err = binary.Write(c.conn, binary.LittleEndian, targetUUIDLen)
	if err != nil {
		return fmt.Errorf("failed to send target UUID length: %w", err)
	}

	// Send target UUID
	_, err = c.conn.Write([]byte(targetUUID))
	if err != nil {
		return fmt.Errorf("failed to send target UUID: %w", err)
	}

	// Send filename length
	fnameLen := uint8(len(filename))
	err = binary.Write(c.conn, binary.LittleEndian, fnameLen)
	if err != nil {
		return fmt.Errorf("failed to send filename length: %w", err)
	}

	// Send filename
	_, err = c.conn.Write([]byte(filename))
	if err != nil {
		return fmt.Errorf("failed to send filename: %w", err)
	}

	// Send file size
	err = binary.Write(c.conn, binary.LittleEndian, uint64(filesize))
	if err != nil {
		return fmt.Errorf("failed to send file size: %w", err)
	}

	// Open and send file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	sent := int64(0)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			_, writeErr := c.conn.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to send file data: %w", writeErr)
			}
			sent += int64(n)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	fmt.Printf("✓ Sent %s to %s (%d bytes)\n", filename, targetUUID, sent)
	return nil
}

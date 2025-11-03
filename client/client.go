package main

import (
    "encoding/binary"
    "fmt"
    "net"
    "time"
)


const (
    putfile uint8 = iota
    listFiles
    streamFile
    ping
    bye
)

// Client represents a connection to the fsend server
type Client struct {
    conn    net.Conn
    address string
}

// NewClient creates a new client instance
func NewClient(address string) *Client {
    return &Client{
        address: address,
    }
}

// Connect establishes a connection to the server
func (c *Client) Connect() error {
    conn, err := net.Dial("tcp", c.address)
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    c.conn = conn
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
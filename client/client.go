package main

import (
    "encoding/binary"
    "fmt"
    "net"
    "time"
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
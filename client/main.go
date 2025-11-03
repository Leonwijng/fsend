package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	putfile uint8 = iota
	listFiles
	streamFile
	ping
	bye
)

func putFile(filePath string, conn net.Conn, bufSize uint32) (err error) {
	fn, err := os.Stat(filePath)
	if err != nil {
		return
	}

	if fn.IsDir() {
		return fmt.Errorf("error: %s is a directory", filePath)
	}

	var (
		fname     = fn.Name()
		fnameSize = len(fname)
		fsize     = fn.Size()
	)

	if fnameSize > 256 {
		return fmt.Errorf("filename is to large, must be smaller than 256 characters")
	}

	err = binary.Write(conn, binary.LittleEndian, putfile)
	if err != nil {
		return
	}

	err = binary.Write(conn, binary.LittleEndian, uint8(fnameSize))
	if err != nil {
		return
	}

	_, err = conn.Write([]byte(fname))
	if err != nil {
		return
	}

	err = binary.Write(conn, binary.LittleEndian, uint64(fsize))
	if err != nil {
		return
	}

	err = binary.Write(conn, binary.LittleEndian, bufSize)
	if err != nil {
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	if bufSize == 0 {
		bufSize = 32 * 1024
	}
	buf := make([]byte, bufSize)
	

	writer := bufio.NewWriter(conn)
	_, err = io.CopyBuffer(writer, f, buf)
	if err != nil {
		return err
	}

	
	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func main() {
    // Create and connect client
    client := NewClient("localhost:3002")
    
    err := client.Connect()
    if err != nil {
        log.Fatalln("Connection failed:", err)
    }
    defer client.Close()
    
    fmt.Println("✓ Connected to server")

    // Test ping
    err = client.Ping()
    if err != nil {
        log.Fatalln("Ping failed:", err)
    }
    fmt.Println("✓ Ping successful")

    // Upload file using existing putFile function
    err = putFile("mascott.png", client.GetConnection(), 1024)
    if err != nil {
        log.Fatalln("File upload failed:", err)
    }
    fmt.Println("✓ File uploaded successfully")
}

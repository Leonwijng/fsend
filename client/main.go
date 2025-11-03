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

	// Upload file
	err = putFile("mascott.png", client.GetConnection(), 1024)
	if err != nil {
		log.Fatalln("File upload failed:", err)
	}
	fmt.Println("✓ File uploaded successfully")

	// List available files
	files, err := client.ListFiles()
	if err != nil {
		log.Fatalln("Failed to list files:", err)
	}

	fmt.Printf("\n✓ Server returned %d files:\n", len(files))
	if len(files) == 0 {
		fmt.Println("  (No files found on server)")
	} else {
		for i, file := range files {
			fmt.Printf("  %d. %s\n", i+1, file)
		}
	}
}

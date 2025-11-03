package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

const filesDir = "files"

// ensureFilesDirectory creates the files directory structure
func ensureFilesDirectory() error {
	return os.MkdirAll(filesDir, 0755)
}

// getUUIDDirectory returns the directory path for a specific UUID
func getUUIDDirectory(uuid string) string {
	return filepath.Join(filesDir, uuid)
}

// ensureUUIDDirectory creates a directory for a specific UUID
func ensureUUIDDirectory(uuid string) error {
	return os.MkdirAll(getUUIDDirectory(uuid), 0755)
}

// listFilesForUUID returns all files for a specific UUID
func listFilesForUUID(uuid string) ([]string, error) {
	dirPath := getUUIDDirectory(uuid)

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return []string{}, nil // No files yet
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// handleListFiles sends the list of available files for the client's UUID
func handleListFiles(conn net.Conn, clientUUID string) error {
	files, err := listFilesForUUID(clientUUID)
	if err != nil {
		fmt.Println("Error listing files:", err)
		binary.Write(conn, binary.LittleEndian, uint32(0))
		return err
	}

	// Send number of files
	err = binary.Write(conn, binary.LittleEndian, uint32(len(files)))
	if err != nil {
		return fmt.Errorf("error sending file count: %w", err)
	}

	// Send each filename
	for _, fname := range files {
		nameLen := uint8(len(fname))
		err = binary.Write(conn, binary.LittleEndian, nameLen)
		if err != nil {
			return fmt.Errorf("error sending filename length: %w", err)
		}

		_, err = conn.Write([]byte(fname))
		if err != nil {
			return fmt.Errorf("error sending filename: %w", err)
		}
	}

	fmt.Printf("✓ Sent %d files to client %s\n", len(files), clientUUID)
	return nil
}

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

// handleStreamFile sends a specific file to the client
func handleStreamFile(conn net.Conn, clientUUID string) error {
	// Read filename length
	var fnameLen uint8
	err := binary.Read(conn, binary.LittleEndian, &fnameLen)
	if err != nil {
		return fmt.Errorf("error reading filename length: %w", err)
	}

	// Read filename
	fnameBytes := make([]byte, fnameLen)
	_, err = conn.Read(fnameBytes)
	if err != nil {
		return fmt.Errorf("error reading filename: %w", err)
	}
	fname := string(fnameBytes)

	// Get file path
	filePath := filepath.Join(getUUIDDirectory(clientUUID), fname)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Send error indicator (filesize = 0)
			binary.Write(conn, binary.LittleEndian, uint64(0))
			return fmt.Errorf("file not found: %s", fname)
		}
		return fmt.Errorf("error checking file: %w", err)
	}

	// Open file
	f, err := os.Open(filePath)
	if err != nil {
		// Send error indicator
		binary.Write(conn, binary.LittleEndian, uint64(0))
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	// Send file size
	fsize := uint64(fileInfo.Size())
	err = binary.Write(conn, binary.LittleEndian, fsize)
	if err != nil {
		return fmt.Errorf("error sending file size: %w", err)
	}

	// Send file data
	buf := make([]byte, 32*1024)
	written, err := f.Read(buf)
	for written > 0 {
		_, err = conn.Write(buf[:written])
		if err != nil {
			return fmt.Errorf("error sending file data: %w", err)
		}
		written, err = f.Read(buf)
	}

	// Close file before deleting
	f.Close()

	// Delete file after successful download
	err = os.Remove(filePath)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to delete file %s: %v\n", fname, err)
	} else {
		fmt.Printf("✓ Sent and deleted file %s for client %s (%d bytes)\n", fname, clientUUID, fsize)
	}

	return nil
}

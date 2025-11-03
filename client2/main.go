package main

import (
	"bufio"
	"encoding/binary"
	"flag"
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

func showMenu() {
	fmt.Println("\n=== fsend Menu ===")
	fmt.Println("1. Upload file (to my storage)")
	fmt.Println("2. Send file to another UUID")
	fmt.Println("3. List my files")
	fmt.Println("4. Download file")
	fmt.Println("5. Ping server")
	fmt.Println("6. Exit")
	fmt.Print("\nChoose option: ")
}

func main() {
	// Check if CLI mode is explicitly requested
	useCLI := flag.Bool("cli", false, "Use CLI mode instead of GUI")
	flag.Parse()

	// Default to GUI mode (when double-clicked)
	if !*useCLI {
		RunGUI()
		return
	}

	// CLI mode
	// Create and connect client
	client, err := NewClient("localhost:3002")
	if err != nil {
		log.Fatalln("Failed to create client:", err)
	}

	err = client.Connect()
	if err != nil {
		log.Fatalln("Connection failed:", err)
	}
	defer client.Close()

	fmt.Println("✓ Connected to server")
	fmt.Printf("Your UUID: %s\n", client.GetUID())

	scanner := bufio.NewScanner(os.Stdin)

	// Interactive menu loop
	for {
		showMenu()

		if !scanner.Scan() {
			break
		}
		choice := scanner.Text()

		switch choice {
		case "1": // Upload to my storage
			fmt.Print("Enter filename to upload: ")
			if !scanner.Scan() {
				break
			}
			filename := scanner.Text()

			err = putFile(filename, client.GetConnection(), 1024)
			if err != nil {
				fmt.Println("❌ Upload failed:", err)
			} else {
				fmt.Println("✓ File uploaded successfully")
			}

		case "2": // Send to another UUID
			fmt.Print("Enter filename to send: ")
			if !scanner.Scan() {
				break
			}
			filename := scanner.Text()

			fmt.Print("Enter target UUID: ")
			if !scanner.Scan() {
				break
			}
			targetUUID := scanner.Text()

			err = client.SendFileToUUID(filename, targetUUID)
			if err != nil {
				fmt.Println("❌ Send failed:", err)
			} else {
				fmt.Printf("✓ File sent to %s\n", targetUUID)
			}

		case "3": // List my files
			files, err := client.ListFiles()
			if err != nil {
				fmt.Println("❌ Failed to list files:", err)
				continue
			}

			fmt.Printf("\n✓ Available files (%d):\n", len(files))
			if len(files) == 0 {
				fmt.Println("  (No files)")
			} else {
				for i, file := range files {
					fmt.Printf("  %d. %s\n", i+1, file)
				}
			}

		case "4": // Download
			files, err := client.ListFiles()
			if err != nil {
				fmt.Println("❌ Failed to list files:", err)
				continue
			}

			if len(files) == 0 {
				fmt.Println("No files available to download")
				continue
			}

			fmt.Println("\nAvailable files:")
			for i, file := range files {
				fmt.Printf("  %d. %s\n", i+1, file)
			}

			fmt.Print("\nEnter file number to download: ")
			if !scanner.Scan() {
				break
			}
			var fileNum int
			_, err = fmt.Sscanf(scanner.Text(), "%d", &fileNum)
			if err != nil || fileNum < 1 || fileNum > len(files) {
				fmt.Println("❌ Invalid file number")
				continue
			}

			downloadName := files[fileNum-1]
			savePath := "downloaded_" + downloadName

			fmt.Printf("Downloading %s...\n", downloadName)
			err = client.DownloadFile(downloadName, savePath)
			if err != nil {
				fmt.Println("❌ Download failed:", err)
			} else {
				fmt.Printf("✓ Saved as %s\n", savePath)
			}

		case "5": // Ping
			err = client.Ping()
			if err != nil {
				fmt.Println("❌ Ping failed:", err)
			} else {
				fmt.Println("✓ Pong!")
			}

		case "6": // Exit
			fmt.Println("Bye!")
			return

		default:
			fmt.Println("❌ Invalid option")
		}
	}
}

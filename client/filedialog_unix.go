//go:build gio && !windows
// +build gio,!windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// On Linux/macOS, fallback to terminal input for file paths
func openFileDialog(title string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s\nEnter file path: ", title)
	path, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(path), nil
}

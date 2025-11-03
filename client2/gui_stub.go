//go:build !gio
// +build !gio

package main

import "fmt"

func RunGUI() {
fmt.Println("GUI mode not available.")
fmt.Println("Rebuild with: go build -tags gio")
}

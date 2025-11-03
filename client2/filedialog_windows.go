//go:build gio && windows
// +build gio,windows

package main

import "github.com/sqweek/dialog"

func openFileDialog(title string) (string, error) {
	return dialog.File().Title(title).Load()
}

package main

import (
	"crypto/rand"
	"fmt"
)

func main() {
	var buf = make([]byte, 4)
	rand.Read(buf)

	fmt.Println(string(buf))
}

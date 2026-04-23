package main

import (
	"bufio"
	"io"
)

func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB maximum buffer to handle very large lines
	return scanner
}

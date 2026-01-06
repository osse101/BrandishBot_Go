package handler

import (
	"bytes"
	"sync"
)

// bufferPool is a pool of bytes.Buffer to reduce allocations during JSON encoding
var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 512)) // Pre-allocate 512 bytes
	},
}

// getBuffer retrieves a buffer from the pool
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer resets the buffer and returns it to the pool
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

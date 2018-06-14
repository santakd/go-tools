// Package io2 is the supplement of the standard library of `io`,
// such as `Close`.
package io2

import (
	"io"

	"github.com/xgfone/go-tools/pools"
)

var (
	bytesPool  = pools.NewBytesPool(4096)
	bufferPool = pools.NewBufferPool()
)

// Close implements the interface with the method Close(), which does not return
// an error.
type Close struct {
	Value io.Closer
}

// Close implements the method Close().
func (c Close) Close() {
	c.Value.Close()
}

// NewClose returns an new Close.
func NewClose(v io.Closer) Close {
	return Close{Value: v}
}

// ReadN reads the data from io.Reader until n bytes or no incoming data
// if n is equal to or less than 0.
func ReadN(r io.Reader, n int64) (v []byte, err error) {
	w := bufferPool.Get()
	err = ReadNWriter(w, r, n)
	v = w.Bytes()
	bufferPool.Put(w)
	return v, err
}

// ReadNWriter is the same as ReadN, but writes the data to the writer
// from the reader.
func ReadNWriter(w io.Writer, r io.Reader, n int64) (err error) {
	buf := bytesPool.Get()

	if n > 0 {
		var m int64
		m, err = io.CopyBuffer(w, io.LimitReader(r, n), buf)
		if m < n && err == nil {
			err = io.EOF
		}
	} else {
		_, err = io.CopyBuffer(w, r, buf)
	}

	bytesPool.Put(buf)
	return
}

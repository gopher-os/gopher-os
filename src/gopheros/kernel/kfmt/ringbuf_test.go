package kfmt

import (
	"bytes"
	"io"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	var (
		buf    bytes.Buffer
		expStr = "the big brown fox jumped over the lazy dog"
		rb     ringBuffer
	)

	t.Run("read/write", func(t *testing.T) {
		rb.wIndex = 0
		rb.rIndex = 0
		n, err := rb.Write([]byte(expStr))
		if err != nil {
			t.Fatal(err)
		}

		if n != len(expStr) {
			t.Fatalf("expected to write %d bytes; wrote %d", len(expStr), n)
		}

		if got := readByteByByte(&buf, &rb); got != expStr {
			t.Fatalf("expected to read %q; got %q", expStr, got)
		}
	})

	t.Run("write moves read pointer", func(t *testing.T) {
		rb.wIndex = ringBufferSize - 1
		rb.rIndex = 0
		_, err := rb.Write([]byte{'!'})
		if err != nil {
			t.Fatal(err)
		}

		if exp := 1; rb.rIndex != exp {
			t.Fatalf("expected write to push rIndex to %d; got %d", exp, rb.rIndex)
		}
	})

	t.Run("wIndex < rIndex", func(t *testing.T) {
		rb.wIndex = ringBufferSize - 2
		rb.rIndex = ringBufferSize - 2
		n, err := rb.Write([]byte(expStr))
		if err != nil {
			t.Fatal(err)
		}

		if n != len(expStr) {
			t.Fatalf("expected to write %d bytes; wrote %d", len(expStr), n)
		}

		if got := readByteByByte(&buf, &rb); got != expStr {
			t.Fatalf("expected to read %q; got %q", expStr, got)
		}
	})

	t.Run("with io.WriteTo", func(t *testing.T) {
		rb.wIndex = ringBufferSize - 2
		rb.rIndex = ringBufferSize - 2
		n, err := rb.Write([]byte(expStr))
		if err != nil {
			t.Fatal(err)
		}

		if n != len(expStr) {
			t.Fatalf("expected to write %d bytes; wrote %d", len(expStr), n)
		}

		var buf bytes.Buffer
		io.Copy(&buf, &rb)

		if got := buf.String(); got != expStr {
			t.Fatalf("expected to read %q; got %q", expStr, got)
		}
	})
}

func readByteByByte(buf *bytes.Buffer, r io.Reader) string {
	buf.Reset()
	var b = make([]byte, 1)
	for {
		_, err := r.Read(b)
		if err == io.EOF {
			break
		}

		buf.Write(b)
	}
	return buf.String()
}

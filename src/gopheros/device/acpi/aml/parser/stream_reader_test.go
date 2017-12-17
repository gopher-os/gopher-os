package parser

import (
	"io"
	"testing"
	"unsafe"
)

func TestAMLStreamReader(t *testing.T) {
	buf := make([]byte, 16)
	for i := 0; i < len(buf); i++ {
		buf[i] = byte(i)
	}

	t.Run("without offset", func(t *testing.T) {
		var r amlStreamReader
		r.Init(
			uintptr(unsafe.Pointer(&buf[0])),
			uint32(len(buf)),
			0,
		)

		if r.EOF() {
			t.Fatal("unexpected EOF")
		}

		if err := r.UnreadByte(); err != errInvalidUnreadByte {
			t.Fatalf("expected errInvalidUnreadByte; got %v", err)
		}

		if _, err := r.LastByte(); err != io.EOF {
			t.Fatalf("unexpected error: %v", err)
		}

		for i := 0; i < len(buf); i++ {
			exp := byte(i)

			next, err := r.PeekByte()
			if err != nil {
				t.Fatal(err)
			}
			if next != exp {
				t.Fatalf("expected PeekByte to return %d; got %d", exp, next)
			}

			next, err = r.ReadByte()
			if err != nil {
				t.Fatal(err)
			}
			if next != exp {
				t.Fatalf("expected ReadByte to return %d; got %d", exp, next)
			}

			last, err := r.LastByte()
			if err != nil {
				t.Fatal(err)
			}
			if last != exp {
				t.Fatalf("expected LastByte to return %d; got %d", exp, last)
			}
		}

		if _, err := r.PeekByte(); err != io.EOF {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := r.ReadByte(); err != io.EOF {
			t.Fatalf("unexpected error: %v", err)
		}
		exp := byte(len(buf) - 1)
		if last, _ := r.LastByte(); last != exp {
			t.Fatalf("expected LastByte to return %d; got %d", exp, last)
		}

	})

	t.Run("with offset", func(t *testing.T) {
		var r amlStreamReader
		r.Init(
			uintptr(unsafe.Pointer(&buf[0])),
			uint32(len(buf)),
			8,
		)

		if r.EOF() {
			t.Fatal("unexpected EOF")
		}

		if exp, got := uint32(8), r.Offset(); got != exp {
			t.Fatalf("expected Offset() to return %d; got %d", exp, got)
		}

		exp := byte(8)
		if next, _ := r.ReadByte(); next != exp {
			t.Fatalf("expected ReadByte to return %d; got %d", exp, next)
		}
	})
}

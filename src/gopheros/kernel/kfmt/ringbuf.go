package kfmt

import "io"

// ringBufferSize defines size of the ring buffer that buffers early Printf
// output. Its default size is selected so it can buffer the contents of a
// standard 80*25 text-mode console. The ring buffer size must always be a
// power of 2.
const ringBufferSize = 2048

// ringBuffer models a ring buffer of size ringBufferSize. This buffer is used
// for capturing the output of Printf before the tty and console systems are
// initialized.
type ringBuffer struct {
	buffer         [ringBufferSize]byte
	rIndex, wIndex int
}

// Write writes len(p) bytes from p to the ringBuffer.
func (rb *ringBuffer) Write(p []byte) (int, error) {
	for _, b := range p {
		rb.buffer[rb.wIndex] = b
		rb.wIndex = (rb.wIndex + 1) & (ringBufferSize - 1)
		if rb.rIndex == rb.wIndex {
			rb.rIndex = (rb.rIndex + 1) & (ringBufferSize - 1)
		}
	}

	return len(p), nil
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0
// <= n <= len(p)) and any error encountered.
func (rb *ringBuffer) Read(p []byte) (n int, err error) {
	switch {
	case rb.rIndex < rb.wIndex:
		// read up to min(wIndex - rIndex, len(p)) bytes
		n = rb.wIndex - rb.rIndex
		if pLen := len(p); pLen < n {
			n = pLen
		}

		copy(p, rb.buffer[rb.rIndex:rb.rIndex+n])
		rb.rIndex += n

		return n, nil
	case rb.rIndex > rb.wIndex:
		// Read up to min(len(buf) - rIndex, len(p)) bytes
		n = len(rb.buffer) - rb.rIndex
		if pLen := len(p); pLen < n {
			n = pLen
		}

		copy(p, rb.buffer[rb.rIndex:rb.rIndex+n])
		rb.rIndex += n

		if rb.rIndex == len(rb.buffer) {
			rb.rIndex = 0
		}

		return n, nil
	default: // rIndex == wIndex
		return 0, io.EOF
	}
}

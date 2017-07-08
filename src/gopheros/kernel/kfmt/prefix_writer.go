package kfmt

import "io"

// PrefixWriter is an io.Writer that wraps another io.Writer and injects a
// prefix at the beginning of each line.
type PrefixWriter struct {
	// A writer where all writes get sent to.
	Sink io.Writer

	// The prefix injected at the beginning of each line.
	Prefix []byte

	bytesAfterPrefix int
}

// Write writes len(p) bytes from p to the underlying data stream and returns
// back the number of bytes written. The PrefixWriter keeps track of the
// beginning of new lines and injects the configured prefix at each new line.
// The injected prefix is not included in the number of written bytes returned
// by this method.
func (w *PrefixWriter) Write(p []byte) (int, error) {
	var (
		written              int
		startIndex, curIndex int
	)

	if w.bytesAfterPrefix == 0 && len(p) != 0 {
		w.Sink.Write(w.Prefix)
	}

	for ; curIndex < len(p); curIndex++ {
		if p[curIndex] == '\n' {
			n, err := w.Sink.Write(p[startIndex : curIndex+1])
			if curIndex+1 != len(p) {
				w.Sink.Write(w.Prefix)
			}
			written += n
			if err != nil {
				return written, err
			}
			w.bytesAfterPrefix = 0
			startIndex = curIndex + 1
		}
	}

	if startIndex < curIndex {
		n, err := w.Sink.Write(p[startIndex:curIndex])
		written += n
		w.bytesAfterPrefix = n
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

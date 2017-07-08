package kfmt

import (
	"bytes"
	"errors"
	"testing"
)

func TestPrefixWriter(t *testing.T) {
	specs := []struct {
		input string
		exp   string
	}{
		{
			"",
			"",
		},
		{
			"\n",
			"prefix: \n",
		},
		{
			"no line break anywhere",
			"prefix: no line break anywhere",
		},
		{
			"line feed at the end\n",
			"prefix: line feed at the end\n",
		},
		{
			"\nthe big brown\nfog jumped\nover the lazy\ndog",
			"prefix: \nprefix: the big brown\nprefix: fog jumped\nprefix: over the lazy\nprefix: dog",
		},
	}

	var (
		buf bytes.Buffer
		w   = PrefixWriter{
			Sink:   &buf,
			Prefix: []byte("prefix: "),
		}
	)

	for specIndex, spec := range specs {
		buf.Reset()
		w.bytesAfterPrefix = 0

		wrote, err := w.Write([]byte(spec.input))
		if err != nil {
			t.Errorf("[spec %d] unexpected error: %v", specIndex, err)
		}

		if expLen := len(spec.input); expLen != wrote {
			t.Errorf("[spec %d] expected writer to write %d bytes; wrote %d", specIndex, expLen, wrote)
		}

		if got := buf.String(); got != spec.exp {
			t.Errorf("[spec %d] expected output:\n%q\ngot:\n%q", specIndex, spec.exp, got)
		}
	}
}

func TestPrefixWriterErrors(t *testing.T) {
	specs := []string{
		"no line break anywhere",
		"\nthe big brown\nfog jumped\nover the lazy\ndog",
	}

	var (
		expErr = errors.New("write failed")
		w      = PrefixWriter{
			Sink:   writerThatAlwaysErrors{expErr},
			Prefix: []byte("prefix: "),
		}
	)

	for specIndex, spec := range specs {
		w.bytesAfterPrefix = 0
		_, err := w.Write([]byte(spec))
		if err != expErr {
			t.Errorf("[spec %d] expected error: %v; got %v", specIndex, expErr, err)
		}
	}
}

type writerThatAlwaysErrors struct {
	err error
}

func (w writerThatAlwaysErrors) Write(_ []byte) (int, error) {
	return 0, w.err
}

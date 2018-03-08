package aml

import (
	"gopheros/kernel"
	"reflect"
	"unsafe"
)

var (
	errInvalidUnreadByte = &kernel.Error{Module: "acpi_aml_parser", Message: "bad call to UnreadByte; stream offset is 0"}
	errInvalidPkgEnd     = &kernel.Error{Module: "acpi_aml_parser", Message: "attempted to set pkgEnd past the end of the stream"}
	errReadPastPkgEnd    = &kernel.Error{Module: "acpi_aml_parser", Message: "attempted to read past pkgEnd"}
)

type amlStreamReader struct {
	offset uint32
	data   []byte
	pkgEnd uint32
}

// Init sets up the reader so it can read up to dataLen bytes from the virtual
// memory address dataAddr. If a non-zero initialOffset is specified, it will
// be used as the current offset in the stream.
func (r *amlStreamReader) Init(dataAddr uintptr, dataLen, initialOffset uint32) {
	// Overlay a byte slice on top of the memory block to be accessed.
	r.data = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  int(dataLen),
		Cap:  int(dataLen),
		Data: dataAddr,
	}))

	r.SetPkgEnd(dataLen)
	r.SetOffset(initialOffset)
}

// EOF returns true if the end of the  pkg has been reached.
func (r *amlStreamReader) EOF() bool {
	return r.offset >= r.pkgEnd
}

func (r *amlStreamReader) SetPkgEnd(pkgEnd uint32) error {
	if pkgEnd > uint32(len(r.data)) {
		return errInvalidPkgEnd
	}

	r.pkgEnd = pkgEnd
	return nil
}

// ReadByte returns the next byte from the stream.
func (r *amlStreamReader) ReadByte() (byte, error) {
	if r.EOF() {
		return 0, errReadPastPkgEnd
	}

	r.offset++
	return r.data[r.offset-1], nil
}

// PeekByte returns the next byte from the stream without advancing the read pointer.
func (r *amlStreamReader) PeekByte() (byte, error) {
	if r.EOF() {
		return 0, errReadPastPkgEnd
	}

	return r.data[r.offset], nil
}

// LastByte returns the last byte read off the stream
func (r *amlStreamReader) LastByte() (byte, error) {
	if r.offset == 0 {
		return 0, errReadPastPkgEnd
	}

	return r.data[r.offset-1], nil
}

// UnreadByte moves back the read pointer by one byte.
func (r *amlStreamReader) UnreadByte() error {
	if r.offset == 0 {
		return errInvalidUnreadByte
	}

	r.offset--
	return nil
}

// Offset returns the current offset.
func (r *amlStreamReader) Offset() uint32 {
	return r.offset
}

// DataPtr returns a pointer to the stream contents at the current stream offset.
func (r *amlStreamReader) DataPtr() uintptr {
	if r.EOF() {
		return 0
	}
	return uintptr(unsafe.Pointer(&r.data[r.offset]))
}

// SetOffset sets the reader offset to the supplied value.
func (r *amlStreamReader) SetOffset(off uint32) {
	if max := uint32(len(r.data)); off > max {
		off = max
	}
	r.offset = off
}

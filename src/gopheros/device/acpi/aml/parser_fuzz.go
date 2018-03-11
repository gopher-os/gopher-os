// +build gofuzz
//
// The following lines contain paths to interesting corpus data and will be
// automatically grepped and copied by the Makefile when fuzzing.
//
//go-fuzz-corpus+=src/gopheros/device/acpi/table/tabletest/DSDT.aml
//go-fuzz-corpus+=src/gopheros/device/acpi/table/tabletest/parser-testsuite-DSDT.aml

package aml

import (
	"gopheros/device/acpi/table"
	"io/ioutil"
	"unsafe"
)

// Fuzz is the driver for go-fuzz. The function must return 1 if the fuzzer
// should increase priority of the given input during subsequent fuzzing (for
// example, the input is lexically correct and was parsed successfully); -1 if
// the input must not be added to corpus even if gives new coverage; and 0
// otherwise; other values are reserved for future use.
func Fuzz(data []byte) int {
	// Setup SDT header pointing to data
	headerLen := unsafe.Sizeof(table.SDTHeader{})
	stream := make([]byte, int(headerLen)+len(data))
	copy(stream[headerLen:], data)

	header := (*table.SDTHeader)(unsafe.Pointer(&stream[0]))
	header.Signature = [4]byte{'D', 'S', 'D', 'T'}
	header.Length = uint32(len(stream))
	header.Revision = 2

	tree := NewObjectTree()
	tree.CreateDefaultScopes(0)
	if err := NewParser(ioutil.Discard, tree).ParseAML(uint8(1), "DSDT", header); err != nil {
		return 0
	}

	return 1
}

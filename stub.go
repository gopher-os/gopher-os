package main

import "github.com/achilleasa/gopher-os/kernel/kmain"

var multibootInfoPtr uintptr

// main makes a dummy call to the actual kernel main entrypoint function. It
// is intentionally defined to prevent the Go compiler from optimizing away the
// real kernel code.
//
// A global variable is passed as an argument to Kmain to prevent the compiler
// from inlining the actual call and removing Kmain from the generated .o file.
func main() {
	kmain.Kmain(multibootInfoPtr, 0, 0)
}

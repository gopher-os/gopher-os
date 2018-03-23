// +build go1.8

package goruntime

import (
	_ "unsafe" // required for go:linkname
)

//go:linkname algInit runtime.alginit
func algInit()

//go:linkname modulesInit runtime.modulesinit
func modulesInit()

//go:linkname typeLinksInit runtime.typelinksinit
func typeLinksInit()

//go:linkname itabsInit runtime.itabsinit
func itabsInit()

//go:linkname mallocInit runtime.mallocinit
func mallocInit()

//go:linkname mSysStatInc runtime.mSysStatInc
func mSysStatInc(*uint64, uintptr)

//go:linkname procResize runtime.procresize
func procResize(int32) uintptr

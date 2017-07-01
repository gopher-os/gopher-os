// Package goruntime contains code for bootstrapping Go runtime features such
// as the memory allocator.
package goruntime

import (
	"gopheros/kernel"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm/allocator"
	"gopheros/kernel/mem/vmm"
	"unsafe"
)

var (
	mapFn                = vmm.Map
	earlyReserveRegionFn = vmm.EarlyReserveRegion
	frameAllocFn         = allocator.AllocFrame
	mallocInitFn         = mallocInit
	algInitFn            = algInit
	modulesInitFn        = modulesInit
	typeLinksInitFn      = typeLinksInit
	itabsInitFn          = itabsInit

	// A seed for the pseudo-random number generator used by getRandomData
	prngSeed = 0xdeadc0de
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

// sysReserve reserves address space without allocating any memory or
// establishing any page mappings.
//
// This function replaces runtime.sysReserve and is required for initializing
// the Go allocator.
//
//go:redirect-from runtime.sysReserve
//go:nosplit
func sysReserve(_ unsafe.Pointer, size uintptr, reserved *bool) unsafe.Pointer {
	regionSize := (mem.Size(size) + mem.PageSize - 1) & ^(mem.PageSize - 1)
	regionStartAddr, err := earlyReserveRegionFn(regionSize)
	if err != nil {
		panic(err)
	}

	*reserved = true
	return unsafe.Pointer(regionStartAddr)
}

// sysMap establishes a copy-on-write mapping for a particular memory region
// that has been reserved previously via a call to sysReserve.
//
// This function replaces runtime.sysReserve and is required for initializing
// the Go allocator.
//
//go:redirect-from runtime.sysMap
//go:nosplit
func sysMap(virtAddr unsafe.Pointer, size uintptr, reserved bool, sysStat *uint64) unsafe.Pointer {
	if !reserved {
		panic("sysMap should only be called with reserved=true")
	}

	// We trust the allocator to call sysMap with an address inside a reserved region.
	regionStartAddr := (uintptr(virtAddr) + uintptr(mem.PageSize-1)) & ^uintptr(mem.PageSize-1)
	regionSize := (mem.Size(size) + mem.PageSize - 1) & ^(mem.PageSize - 1)
	pageCount := regionSize >> mem.PageShift

	mapFlags := vmm.FlagPresent | vmm.FlagNoExecute | vmm.FlagCopyOnWrite
	for page := vmm.PageFromAddress(regionStartAddr); pageCount > 0; pageCount, page = pageCount-1, page+1 {
		if err := mapFn(page, vmm.ReservedZeroedFrame, mapFlags); err != nil {
			return unsafe.Pointer(uintptr(0))
		}
	}

	mSysStatInc(sysStat, uintptr(regionSize))
	return unsafe.Pointer(regionStartAddr)
}

// sysAlloc reserves enough phsysical frames to satisfy the allocation request
// and establishes a contiguous virtual page mapping for them returning back
// the pointer to the virtual region start.
//
// This function replaces runtime.sysMap and is required for initializing the
// Go allocator.
//
//go:redirect-from runtime.sysAlloc
//go:nosplit
func sysAlloc(size uintptr, sysStat *uint64) unsafe.Pointer {
	regionSize := (mem.Size(size) + mem.PageSize - 1) & ^(mem.PageSize - 1)
	regionStartAddr, err := earlyReserveRegionFn(regionSize)
	if err != nil {
		return unsafe.Pointer(uintptr(0))
	}

	mapFlags := vmm.FlagPresent | vmm.FlagNoExecute | vmm.FlagRW
	pageCount := regionSize >> mem.PageShift
	for page := vmm.PageFromAddress(regionStartAddr); pageCount > 0; pageCount, page = pageCount-1, page+1 {
		frame, err := frameAllocFn()
		if err != nil {
			return unsafe.Pointer(uintptr(0))
		}

		if err = mapFn(page, frame, mapFlags); err != nil {
			return unsafe.Pointer(uintptr(0))
		}
	}

	mSysStatInc(sysStat, uintptr(regionSize))
	return unsafe.Pointer(regionStartAddr)
}

// nanotime returns a monotonically increasing clock value. This is a dummy
// implementation and will be replaced when the timekeeper package is
// implemented.
//
// This function replaces runtime.nanotime and is invoked by the Go allocator
// when a span allocation is performed.
//
//go:redirect-from runtime.nanotime
//go:nosplit
func nanotime() uint64 {
	// Use a dummy loop to prevent the compiler from inlining this function.
	for i := 0; i < 100; i++ {
	}
	return 1
}

// getRandomData populates the given slice with random data. The implementation
// is the runtime package reads a random stream from /dev/random but since this
// is not available, we use a prng instead.
//
//go:redirect-from runtime.getRandomData
func getRandomData(r []byte) {
	for i := 0; i < len(r); i++ {
		prngSeed = (prngSeed * 58321) + 11113
		r[i] = byte((prngSeed >> 16) & 255)
	}
}

// Init enables support for various Go runtime features. After a call to init
// the following runtime features become available for use:
//  - heap memory allocation (new, make e.t.c)
//  - map primitives
//  - interfaces
func Init() *kernel.Error {
	mallocInitFn()
	algInitFn()       // setup hash implementation for map keys
	modulesInitFn()   // provides activeModules
	typeLinksInitFn() // uses maps, activeModules
	itabsInitFn()     // uses activeModules

	return nil
}

func init() {
	// Dummy calls so the compiler does not optimize away the functions in
	// this file.
	var (
		reserved bool
		stat     uint64
		zeroPtr  = unsafe.Pointer(uintptr(0))
	)

	sysReserve(zeroPtr, 0, &reserved)
	sysMap(zeroPtr, 0, reserved, &stat)
	sysAlloc(0, &stat)
	getRandomData(nil)
	stat = nanotime()
}

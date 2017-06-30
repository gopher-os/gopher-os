// Package goruntime contains code for bootstrapping Go runtime features such
// as the memory allocator.
package goruntime

import (
	"unsafe"

	"github.com/achilleasa/gopher-os/kernel/mem"
	"github.com/achilleasa/gopher-os/kernel/mem/pmm/allocator"
	"github.com/achilleasa/gopher-os/kernel/mem/vmm"
)

var (
	mapFn                = vmm.Map
	earlyReserveRegionFn = vmm.EarlyReserveRegion
	frameAllocFn         = allocator.AllocFrame
)

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
}

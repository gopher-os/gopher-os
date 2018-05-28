package pmm

import (
	"gopheros/kernel"
	"gopheros/kernel/mm"
)

var (
	// bootMemAllocator is the page allocator used when the kernel boots.
	// It is used to bootstrap the bitmap allocator which is used for all
	// page allocations while the kernel runs.
	bootMemAllocator BootMemAllocator

	// bitmapAllocator is the standard allocator used by the kernel.
	bitmapAllocator BitmapAllocator
)

// Init sets up the kernel physical memory allocation sub-system.
func Init(kernelStart, kernelEnd uintptr) *kernel.Error {
	bootMemAllocator.init(kernelStart, kernelEnd)
	bootMemAllocator.printMemoryMap()
	mm.SetFrameAllocator(earlyAllocFrame)

	// Using the bootMemAllocator bootstrap the bitmap allocator
	if err := bitmapAllocator.init(); err != nil {
		return err
	}
	mm.SetFrameAllocator(bitmapAllocFrame)

	return nil
}

func earlyAllocFrame() (mm.Frame, *kernel.Error) {
	return bootMemAllocator.AllocFrame()
}

func bitmapAllocFrame() (mm.Frame, *kernel.Error) {
	return bitmapAllocator.AllocFrame()
}

package vmm

import (
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/irq"
	"gopheros/kernel/kfmt"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
	"unsafe"
)

var (
	// frameAllocator points to a frame allocator function registered using
	// SetFrameAllocator.
	frameAllocator FrameAllocatorFn

	// the following functions are mocked by tests and are automatically
	// inlined by the compiler.
	handleExceptionWithCodeFn = irq.HandleExceptionWithCode
	readCR2Fn                 = cpu.ReadCR2
	translateFn               = Translate
	visitElfSectionsFn        = multiboot.VisitElfSections

	errUnrecoverableFault = &kernel.Error{Module: "vmm", Message: "page/gpf fault"}
)

// FrameAllocatorFn is a function that can allocate physical frames.
type FrameAllocatorFn func() (pmm.Frame, *kernel.Error)

// SetFrameAllocator registers a frame allocator function that will be used by
// the vmm code when new physical frames need to be allocated.
func SetFrameAllocator(allocFn FrameAllocatorFn) {
	frameAllocator = allocFn
}

func pageFaultHandler(errorCode uint64, frame *irq.Frame, regs *irq.Regs) {
	var (
		faultAddress = uintptr(readCR2Fn())
		faultPage    = PageFromAddress(faultAddress)
		pageEntry    *pageTableEntry
	)

	// Lookup entry for the page where the fault occurred
	walk(faultPage.Address(), func(pteLevel uint8, pte *pageTableEntry) bool {
		nextIsPresent := pte.HasFlags(FlagPresent)

		if pteLevel == pageLevels-1 && nextIsPresent {
			pageEntry = pte
		}

		// Abort walk if the next page table entry is missing
		return nextIsPresent
	})

	// CoW is supported for RO pages with the CoW flag set
	if pageEntry != nil && !pageEntry.HasFlags(FlagRW) && pageEntry.HasFlags(FlagCopyOnWrite) {
		var (
			copy    pmm.Frame
			tmpPage Page
			err     *kernel.Error
		)

		if copy, err = frameAllocator(); err != nil {
			nonRecoverablePageFault(faultAddress, errorCode, frame, regs, err)
		} else if tmpPage, err = mapTemporaryFn(copy); err != nil {
			nonRecoverablePageFault(faultAddress, errorCode, frame, regs, err)
		} else {
			// Copy page contents, mark as RW and remove CoW flag
			mem.Memcopy(faultPage.Address(), tmpPage.Address(), mem.PageSize)
			unmapFn(tmpPage)

			// Update mapping to point to the new frame, flag it as RW and
			// remove the CoW flag
			pageEntry.ClearFlags(FlagCopyOnWrite)
			pageEntry.SetFlags(FlagPresent | FlagRW)
			pageEntry.SetFrame(copy)
			flushTLBEntryFn(faultPage.Address())

			// Fault recovered; retry the instruction that caused the fault
			return
		}
	}

	nonRecoverablePageFault(faultAddress, errorCode, frame, regs, errUnrecoverableFault)
}

func nonRecoverablePageFault(faultAddress uintptr, errorCode uint64, frame *irq.Frame, regs *irq.Regs, err *kernel.Error) {
	kfmt.Printf("\nPage fault while accessing address: 0x%16x\nReason: ", faultAddress)
	switch {
	case errorCode == 0:
		kfmt.Printf("read from non-present page")
	case errorCode == 1:
		kfmt.Printf("page protection violation (read)")
	case errorCode == 2:
		kfmt.Printf("write to non-present page")
	case errorCode == 3:
		kfmt.Printf("page protection violation (write)")
	case errorCode == 4:
		kfmt.Printf("page-fault in user-mode")
	case errorCode == 8:
		kfmt.Printf("page table has reserved bit set")
	case errorCode == 16:
		kfmt.Printf("instruction fetch")
	default:
		kfmt.Printf("unknown")
	}

	kfmt.Printf("\n\nRegisters:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panic(err)
}

func generalProtectionFaultHandler(_ uint64, frame *irq.Frame, regs *irq.Regs) {
	kfmt.Printf("\nGeneral protection fault while accessing address: 0x%x\n", readCR2Fn())
	kfmt.Printf("Registers:\n")
	regs.Print()
	frame.Print()

	// TODO: Revisit this when user-mode tasks are implemented
	panic(errUnrecoverableFault)
}

// reserveZeroedFrame reserves a physical frame to be used together with
// FlagCopyOnWrite for lazy allocation requests.
func reserveZeroedFrame() *kernel.Error {
	var (
		err      *kernel.Error
		tempPage Page
	)

	if ReservedZeroedFrame, err = frameAllocator(); err != nil {
		return err
	} else if tempPage, err = mapTemporaryFn(ReservedZeroedFrame); err != nil {
		return err
	}
	mem.Memset(tempPage.Address(), 0, mem.PageSize)
	unmapFn(tempPage)

	// From this point on, ReservedZeroedFrame cannot be mapped with a RW flag
	protectReservedZeroedPage = true
	return nil
}

// Init initializes the vmm system, creates a granular PDT for the kernel and
// installs paging-related exception handlers.
func Init(kernelPageOffset uintptr) *kernel.Error {
	if err := setupPDTForKernel(kernelPageOffset); err != nil {
		return err
	}

	if err := reserveZeroedFrame(); err != nil {
		return err
	}

	handleExceptionWithCodeFn(irq.PageFaultException, pageFaultHandler)
	handleExceptionWithCodeFn(irq.GPFException, generalProtectionFaultHandler)
	return nil
}

// setupPDTForKernel queries the multiboot package for the ELF sections that
// correspond to the loaded kernel image and establishes a new granular PDT for
// the kernel's VMA using the appropriate flags (e.g. NX for data sections, RW
// for writable sections e.t.c).
func setupPDTForKernel(kernelPageOffset uintptr) *kernel.Error {
	var pdt PageDirectoryTable

	// Allocate frame for the page directory and initialize it
	pdtFrame, err := frameAllocator()
	if err != nil {
		return err
	}

	if err = pdt.Init(pdtFrame); err != nil {
		return err
	}

	// Query the ELF sections of the kernel image and establish mappings
	// for each one using the appropriate flags
	var visitor = func(_ string, secFlags multiboot.ElfSectionFlag, secAddress uintptr, secSize uint64) {
		// Bail out if we have encountered an error; also ignore sections
		// not using the kernel's VMA
		if err != nil || secAddress < kernelPageOffset {
			return
		}

		flags := FlagPresent

		if (secFlags & multiboot.ElfSectionExecutable) == 0 {
			flags |= FlagNoExecute
		}

		if (secFlags & multiboot.ElfSectionWritable) != 0 {
			flags |= FlagRW
		}

		// Map the start and end VMA addresses for the section contents
		// into a start and end (inclusive) page number. To figure out
		// the physical start frame we just need to subtract the
		// kernel's VMA offset from the virtual address and round that
		// down to the nearest frame number.
		curPage := PageFromAddress(secAddress)
		lastPage := PageFromAddress(secAddress + uintptr(secSize-1))
		curFrame := pmm.Frame((secAddress - kernelPageOffset) >> mem.PageShift)
		for ; curPage <= lastPage; curFrame, curPage = curFrame+1, curPage+1 {
			if err = pdt.Map(curPage, curFrame, flags); err != nil {
				return
			}
		}
	}

	// Use the noescape hack to prevent the compiler from leaking the visitor
	// function literal to the heap.
	visitElfSectionsFn(
		*(*multiboot.ElfSectionVisitor)(noEscape(unsafe.Pointer(&visitor))),
	)

	// If an error occurred while maping the ELF sections bail out
	if err != nil {
		return err
	}

	// Ensure that any pages mapped by the memory allocator using
	// EarlyReserveRegion are copied to the new page directory.
	for rsvAddr := earlyReserveLastUsed; rsvAddr < tempMappingAddr; rsvAddr += uintptr(mem.PageSize) {
		page := PageFromAddress(rsvAddr)

		frameAddr, err := translateFn(rsvAddr)
		if err != nil {
			return err
		}

		if err = pdt.Map(page, pmm.Frame(frameAddr>>mem.PageShift), FlagPresent|FlagRW); err != nil {
			return err
		}
	}

	// Activate the new PDT. After this point, the identify mapping for the
	// physical memory addresses where the kernel is loaded becomes invalid.
	pdt.Activate()

	return nil
}

// noEscape hides a pointer from escape analysis. This function is copied over
// from runtime/stubs.go
//go:nosplit
func noEscape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

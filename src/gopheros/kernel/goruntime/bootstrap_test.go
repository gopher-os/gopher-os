package goruntime

import (
	"gopheros/kernel"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
	"gopheros/kernel/mem/pmm/allocator"
	"gopheros/kernel/mem/vmm"
	"reflect"
	"testing"
	"unsafe"
)

func TestSysReserve(t *testing.T) {
	defer func() {
		earlyReserveRegionFn = vmm.EarlyReserveRegion
	}()
	var reserved bool

	t.Run("success", func(t *testing.T) {
		specs := []struct {
			reqSize       mem.Size
			expRegionSize mem.Size
		}{
			// exact multiple of page size
			{100 << mem.PageShift, 100 << mem.PageShift},
			// size should be rounded up to nearest page size
			{2*mem.PageSize - 1, 2 * mem.PageSize},
		}

		for specIndex, spec := range specs {
			earlyReserveRegionFn = func(rsvSize mem.Size) (uintptr, *kernel.Error) {
				if rsvSize != spec.expRegionSize {
					t.Errorf("[spec %d] expected reservation size to be %d; got %d", specIndex, spec.expRegionSize, rsvSize)
				}

				return 0xbadf00d, nil
			}

			ptr := sysReserve(nil, uintptr(spec.reqSize), &reserved)
			if uintptr(ptr) == 0 {
				t.Errorf("[spec %d] sysReserve returned 0", specIndex)
				continue
			}
		}
	})

	t.Run("fail", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("expected sysReserve to panic")
			}
		}()

		earlyReserveRegionFn = func(rsvSize mem.Size) (uintptr, *kernel.Error) {
			return 0, &kernel.Error{Module: "test", Message: "consumed available address space"}
		}

		sysReserve(nil, uintptr(0xf00), &reserved)
	})
}

func TestSysMap(t *testing.T) {
	defer func() {
		earlyReserveRegionFn = vmm.EarlyReserveRegion
		mapFn = vmm.Map
	}()

	t.Run("success", func(t *testing.T) {
		specs := []struct {
			reqAddr         uintptr
			reqSize         mem.Size
			expRsvAddr      uintptr
			expMapCallCount int
		}{
			// exact multiple of page size
			{100 << mem.PageShift, 4 * mem.PageSize, 100 << mem.PageShift, 4},
			// address should be rounded up to nearest page size
			{(100 << mem.PageShift) + 1, 4 * mem.PageSize, 101 << mem.PageShift, 4},
			// size should be rounded up to nearest page size
			{1 << mem.PageShift, (4 * mem.PageSize) + 1, 1 << mem.PageShift, 5},
		}

		for specIndex, spec := range specs {
			var (
				sysStat      uint64
				mapCallCount int
			)
			mapFn = func(_ vmm.Page, _ pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
				expFlags := vmm.FlagPresent | vmm.FlagCopyOnWrite | vmm.FlagNoExecute
				if flags != expFlags {
					t.Errorf("[spec %d] expected map flags to be %d; got %d", specIndex, expFlags, flags)
				}
				mapCallCount++
				return nil
			}

			rsvPtr := sysMap(unsafe.Pointer(spec.reqAddr), uintptr(spec.reqSize), true, &sysStat)
			if got := uintptr(rsvPtr); got != spec.expRsvAddr {
				t.Errorf("[spec %d] expected mapped address 0x%x; got 0x%x", specIndex, spec.expRsvAddr, got)
			}

			if mapCallCount != spec.expMapCallCount {
				t.Errorf("[spec %d] expected vmm.Map call count to be %d; got %d", specIndex, spec.expMapCallCount, mapCallCount)
			}

			if exp := uint64(spec.expMapCallCount << mem.PageShift); sysStat != exp {
				t.Errorf("[spec %d] expected stat counter to be %d; got %d", specIndex, exp, sysStat)
			}
		}
	})

	t.Run("map fails", func(t *testing.T) {
		mapFn = func(_ vmm.Page, _ pmm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error {
			return &kernel.Error{Module: "test", Message: "map failed"}
		}

		var sysStat uint64
		if got := sysMap(unsafe.Pointer(uintptr(0xbadf00d)), 1, true, &sysStat); got != unsafe.Pointer(uintptr(0)) {
			t.Fatalf("expected sysMap to return 0x0 if Map returns an error; got 0x%x", uintptr(got))
		}
	})

	t.Run("panic if not reserved", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("expected sysMap to panic")
			}
		}()

		sysMap(nil, 0, false, nil)
	})
}

func TestSysAlloc(t *testing.T) {
	defer func() {
		earlyReserveRegionFn = vmm.EarlyReserveRegion
		mapFn = vmm.Map
		memsetFn = mem.Memset
		frameAllocFn = allocator.AllocFrame
	}()

	t.Run("success", func(t *testing.T) {
		specs := []struct {
			reqSize         mem.Size
			expMapCallCount int
		}{
			// exact multiple of page size
			{4 * mem.PageSize, 4},
			// round up to nearest page size
			{(4 * mem.PageSize) + 1, 5},
		}

		expRegionStartAddr := uintptr(10 * mem.PageSize)
		earlyReserveRegionFn = func(_ mem.Size) (uintptr, *kernel.Error) {
			return expRegionStartAddr, nil
		}

		frameAllocFn = func() (pmm.Frame, *kernel.Error) {
			return pmm.Frame(0), nil
		}

		for specIndex, spec := range specs {
			var (
				sysStat         uint64
				mapCallCount    int
				memsetCallCount int
			)

			memsetFn = func(_ uintptr, _ byte, _ mem.Size) {
				memsetCallCount++
			}

			mapFn = func(_ vmm.Page, _ pmm.Frame, flags vmm.PageTableEntryFlag) *kernel.Error {
				expFlags := vmm.FlagPresent | vmm.FlagNoExecute | vmm.FlagRW
				if flags != expFlags {
					t.Errorf("[spec %d] expected map flags to be %d; got %d", specIndex, expFlags, flags)
				}
				mapCallCount++
				return nil
			}

			if got := sysAlloc(uintptr(spec.reqSize), &sysStat); uintptr(got) != expRegionStartAddr {
				t.Errorf("[spec %d] expected sysAlloc to return address 0x%x; got 0x%x", specIndex, expRegionStartAddr, uintptr(got))
			}

			if mapCallCount != spec.expMapCallCount {
				t.Errorf("[spec %d] expected vmm.Map call count to be %d; got %d", specIndex, spec.expMapCallCount, mapCallCount)
			}

			// sysAlloc should perform the same number of memset calls as map calls
			if memsetCallCount != spec.expMapCallCount {
				t.Errorf("[spec %d] expected mem.Memset call count to be %d; got %d", specIndex, spec.expMapCallCount, memsetCallCount)
			}

			if exp := uint64(spec.expMapCallCount << mem.PageShift); sysStat != exp {
				t.Errorf("[spec %d] expected stat counter to be %d; got %d", specIndex, exp, sysStat)
			}
		}
	})

	t.Run("earlyReserveRegion fails", func(t *testing.T) {
		earlyReserveRegionFn = func(rsvSize mem.Size) (uintptr, *kernel.Error) {
			return 0, &kernel.Error{Module: "test", Message: "consumed available address space"}
		}

		var sysStat uint64
		if got := sysAlloc(1, &sysStat); got != unsafe.Pointer(uintptr(0)) {
			t.Fatalf("expected sysAlloc to return 0x0 if EarlyReserveRegion returns an error; got 0x%x", uintptr(got))
		}
	})

	t.Run("frame allocation fails", func(t *testing.T) {
		expRegionStartAddr := uintptr(10 * mem.PageSize)
		earlyReserveRegionFn = func(rsvSize mem.Size) (uintptr, *kernel.Error) {
			return expRegionStartAddr, nil
		}

		frameAllocFn = func() (pmm.Frame, *kernel.Error) {
			return pmm.InvalidFrame, &kernel.Error{Module: "test", Message: "out of memory"}
		}

		var sysStat uint64
		if got := sysAlloc(1, &sysStat); got != unsafe.Pointer(uintptr(0)) {
			t.Fatalf("expected sysAlloc to return 0x0 if AllocFrame returns an error; got 0x%x", uintptr(got))
		}
	})

	t.Run("map fails", func(t *testing.T) {
		expRegionStartAddr := uintptr(10 * mem.PageSize)
		earlyReserveRegionFn = func(rsvSize mem.Size) (uintptr, *kernel.Error) {
			return expRegionStartAddr, nil
		}

		frameAllocFn = func() (pmm.Frame, *kernel.Error) {
			return pmm.Frame(0), nil
		}

		mapFn = func(_ vmm.Page, _ pmm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error {
			return &kernel.Error{Module: "test", Message: "map failed"}
		}

		var sysStat uint64
		if got := sysAlloc(1, &sysStat); got != unsafe.Pointer(uintptr(0)) {
			t.Fatalf("expected sysAlloc to return 0x0 if AllocFrame returns an error; got 0x%x", uintptr(got))
		}
	})
}

func TestGetRandomData(t *testing.T) {
	sample1 := make([]byte, 128)
	sample2 := make([]byte, 128)

	getRandomData(sample1)
	getRandomData(sample2)

	if reflect.DeepEqual(sample1, sample2) {
		t.Fatal("expected getRandomData to return different values for each invocation")
	}
}

func TestInit(t *testing.T) {
	defer func() {
		mallocInitFn = mallocInit
		algInitFn = algInit
		modulesInitFn = modulesInit
		typeLinksInitFn = typeLinksInit
		itabsInitFn = itabsInit
		initGoPackagesFn = initGoPackages
		procResizeFn = procResize
	}()

	mallocInitFn = func() {}
	algInitFn = func() {}
	modulesInitFn = func() {}
	typeLinksInitFn = func() {}
	itabsInitFn = func() {}
	initGoPackagesFn = func() {}
	procResizeFn = func(_ int32) uintptr { return 0 }
	if err := Init(); err != nil {
		t.Fatal(t)
	}
}

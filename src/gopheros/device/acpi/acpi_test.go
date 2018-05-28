package acpi

import (
	"gopheros/device/acpi/table"
	"gopheros/kernel"
	"gopheros/kernel/mm"
	"gopheros/kernel/mm/vmm"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

var (
	dsdtSignature = "DSDT"
)

func TestProbe(t *testing.T) {
	defer func(rsdpLow, rsdpHi, rsdpAlign uintptr) {
		mapFn = vmm.Map
		unmapFn = vmm.Unmap
		rsdpLocationLow = rsdpLow
		rsdpLocationHi = rsdpHi
		rsdpAlignment = rsdpAlign
	}(rsdpLocationLow, rsdpLocationHi, rsdpAlignment)

	t.Run("ACPI1", func(t *testing.T) {
		mapFn = func(_ mm.Page, _ mm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error { return nil }
		unmapFn = func(_ mm.Page) *kernel.Error { return nil }

		// Allocate space for 2 descriptors; leave the first entry
		// blank to test that locateRSDT will jump over it and populate
		// the second descriptor
		sizeofRSDP := unsafe.Sizeof(table.RSDPDescriptor{})
		buf := make([]byte, 2*sizeofRSDP)
		rsdpHeader := (*table.RSDPDescriptor)(unsafe.Pointer(&buf[sizeofRSDP]))
		rsdpHeader.Signature = rsdpSignature
		rsdpHeader.Revision = acpiRev1
		rsdpHeader.RSDTAddr = 0xbadf00
		rsdpHeader.Checksum = -calcChecksum(uintptr(unsafe.Pointer(rsdpHeader)), uintptr(sizeofRSDP))

		rsdpLocationLow = uintptr(unsafe.Pointer(&buf[0]))
		rsdpLocationHi = uintptr(unsafe.Pointer(&buf[2*sizeofRSDP-1]))
		// As we cannot ensure 16-byte alignment for our buffer we need to override the
		// alignment so we scan all bytes in the buffer for the descriptor signature
		rsdpAlignment = 1

		drv := probeForACPI()
		if drv == nil {
			t.Fatal("ACPI probe failed")
		}

		drv.DriverName()
		drv.DriverVersion()

		acpiDrv := drv.(*acpiDriver)

		if acpiDrv.rsdtAddr != uintptr(rsdpHeader.RSDTAddr) {
			t.Fatalf("expected probed RSDT address to be 0x%x; got 0x%x", uintptr(rsdpHeader.RSDTAddr), acpiDrv.rsdtAddr)
		}

		if exp := false; acpiDrv.useXSDT != exp {
			t.Fatal("expected probe to locate the RSDT and not the XSDT")
		}
	})

	t.Run("ACPI2+", func(t *testing.T) {
		mapFn = func(_ mm.Page, _ mm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error { return nil }
		unmapFn = func(_ mm.Page) *kernel.Error { return nil }

		// Allocate space for 2 descriptors; leave the first entry
		// blank to test that locateRSDT will jump over it and populate
		// the second descriptor
		sizeofRSDP := unsafe.Sizeof(table.RSDPDescriptor{})
		sizeofExtRSDP := unsafe.Sizeof(table.ExtRSDPDescriptor{})
		buf := make([]byte, 2*sizeofExtRSDP)
		rsdpHeader := (*table.ExtRSDPDescriptor)(unsafe.Pointer(&buf[sizeofExtRSDP]))
		rsdpHeader.Signature = rsdpSignature
		rsdpHeader.Revision = acpiRev2Plus
		rsdpHeader.RSDTAddr = 0xbadf00 // we should ignore this and use XSDT instrad
		rsdpHeader.Checksum = -calcChecksum(uintptr(unsafe.Pointer(rsdpHeader)), uintptr(sizeofRSDP))

		rsdpHeader.XSDTAddr = 0xc0ffee
		rsdpHeader.ExtendedChecksum = -calcChecksum(uintptr(unsafe.Pointer(rsdpHeader)), uintptr(sizeofExtRSDP))

		rsdpLocationLow = uintptr(unsafe.Pointer(&buf[0]))
		rsdpLocationHi = uintptr(unsafe.Pointer(&buf[2*sizeofExtRSDP-1]))
		// As we cannot ensure 16-byte alignment for our buffer we need to override the
		// alignment so we scan all bytes in the buffer for the descriptor signature
		rsdpAlignment = 1

		drv := probeForACPI()
		if drv == nil {
			t.Fatal("ACPI probe failed")
		}

		acpiDrv := drv.(*acpiDriver)

		if acpiDrv.rsdtAddr != uintptr(rsdpHeader.XSDTAddr) {
			t.Fatalf("expected probed RSDT address to be 0x%x; got 0x%x", uintptr(rsdpHeader.XSDTAddr), acpiDrv.rsdtAddr)
		}

		if exp := true; acpiDrv.useXSDT != exp {
			t.Fatal("expected probe to locate the XSDT and not the RSDT")
		}
	})

	t.Run("RSDP ACPI1 checksum mismatch", func(t *testing.T) {
		mapFn = func(_ mm.Page, _ mm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error { return nil }
		unmapFn = func(_ mm.Page) *kernel.Error { return nil }

		sizeofRSDP := unsafe.Sizeof(table.RSDPDescriptor{})
		buf := make([]byte, sizeofRSDP)
		rsdpHeader := (*table.RSDPDescriptor)(unsafe.Pointer(&buf[0]))
		rsdpHeader.Signature = rsdpSignature
		rsdpHeader.Revision = acpiRev1

		// Set wrong checksum
		rsdpHeader.Checksum = 0

		// As we cannot ensure 16-byte alignment for our buffer we need to override the
		// alignment so we scan all bytes in the buffer for the descriptor signature
		rsdpLocationLow = uintptr(unsafe.Pointer(&buf[0]))
		rsdpLocationHi = uintptr(unsafe.Pointer(&buf[sizeofRSDP-1]))
		rsdpAlignment = 1

		drv := probeForACPI()
		if drv != nil {
			t.Fatal("expected ACPI probe to fail")
		}
	})

	t.Run("RSDP ACPI2+ checksum mismatch", func(t *testing.T) {
		mapFn = func(_ mm.Page, _ mm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error { return nil }
		unmapFn = func(_ mm.Page) *kernel.Error { return nil }

		sizeofExtRSDP := unsafe.Sizeof(table.ExtRSDPDescriptor{})
		buf := make([]byte, sizeofExtRSDP)
		rsdpHeader := (*table.ExtRSDPDescriptor)(unsafe.Pointer(&buf[0]))
		rsdpHeader.Signature = rsdpSignature
		rsdpHeader.Revision = acpiRev2Plus

		// Set wrong checksum for extended rsdp
		rsdpHeader.ExtendedChecksum = 0

		// As we cannot ensure 16-byte alignment for our buffer we need to override the
		// alignment so we scan all bytes in the buffer for the descriptor signature
		rsdpLocationLow = uintptr(unsafe.Pointer(&buf[0]))
		rsdpLocationHi = uintptr(unsafe.Pointer(&buf[sizeofExtRSDP-1]))
		rsdpAlignment = 1

		drv := probeForACPI()
		if drv != nil {
			t.Fatal("expected ACPI probe to fail")
		}
	})

	t.Run("error mapping rsdp memory block", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "vmm.Map failed"}
		mapFn = func(_ mm.Page, _ mm.Frame, _ vmm.PageTableEntryFlag) *kernel.Error { return expErr }
		unmapFn = func(_ mm.Page) *kernel.Error { return nil }

		drv := probeForACPI()
		if drv != nil {
			t.Fatal("expected ACPI probe to fail")
		}
	})
}

func TestDriverInit(t *testing.T) {
	defer func() {
		identityMapFn = vmm.IdentityMapRegion
	}()

	t.Run("success", func(t *testing.T) {
		rsdtAddr, _ := genTestRDST(t, acpiRev2Plus)
		identityMapFn = func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
			return mm.Page(frame), nil
		}

		drv := &acpiDriver{
			rsdtAddr: rsdtAddr,
			useXSDT:  true,
		}

		if err := drv.DriverInit(os.Stderr); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("map errors in enumerateTables", func(t *testing.T) {
		rsdtAddr, tableList := genTestRDST(t, acpiRev2Plus)

		var (
			expErr    = &kernel.Error{Module: "test", Message: "vmm.Map failed"}
			callCount int
		)

		drv := &acpiDriver{
			rsdtAddr: rsdtAddr,
			useXSDT:  true,
		}

		specs := []func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error){
			func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
				// fail while trying to map RSDT
				return 0, expErr
			},
			func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
				// fail while trying to map any other ACPI table
				callCount++
				if callCount > 2 {
					return 0, expErr
				}
				return mm.Page(frame), nil
			},
			func(frame mm.Frame, size uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
				// fail while trying to map DSDT
				for _, header := range tableList {
					if header.Length == uint32(size) && string(header.Signature[:]) == dsdtSignature {
						return 0, expErr
					}
				}
				return mm.Page(frame), nil
			},
		}

		// Test map errors for all map calls in enumerateTables
		for specIndex, spec := range specs {
			identityMapFn = spec
			if err := drv.DriverInit(os.Stderr); err != expErr {
				t.Errorf("[spec %d]; expected to get an error\n", specIndex)
			}
		}
	})

}

func TestEnumerateTables(t *testing.T) {
	defer func() {
		identityMapFn = vmm.IdentityMapRegion
	}()

	var expTables = []string{"SSDT", "APIC", "FACP", "DSDT"}

	t.Run("ACPI1", func(t *testing.T) {
		rsdtAddr, tableList := genTestRDST(t, acpiRev1)

		identityMapFn = func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
			// The frame encodes the table index we need to lookup (see genTestRDST)
			nextTableIndex := int(frame)
			if nextTableIndex >= len(tableList) {
				// This is the RSDT
				return mm.Page(frame), nil
			}

			header := tableList[nextTableIndex]
			return mm.PageFromAddress(uintptr(unsafe.Pointer(header))), nil
		}

		drv := &acpiDriver{
			rsdtAddr: rsdtAddr,
			useXSDT:  false,
		}

		if err := drv.enumerateTables(os.Stderr); err != nil {
			t.Fatal(err)
		}

		if exp, got := len(expTables), len(drv.tableMap); got != exp {
			t.Fatalf("expected enumerateTables to discover %d tables; got %d\n", exp, got)
		}

		for _, tableName := range expTables {
			if drv.tableMap[tableName] == nil {
				t.Fatalf("expected enumerateTables to discover table %q", tableName)
			}
		}

		drv.printTableInfo(os.Stderr)
	})

	t.Run("ACPI2+", func(t *testing.T) {
		rsdtAddr, _ := genTestRDST(t, acpiRev2Plus)
		identityMapFn = func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
			return mm.Page(frame), nil
		}

		drv := &acpiDriver{
			rsdtAddr: rsdtAddr,
			useXSDT:  true,
		}

		if err := drv.enumerateTables(os.Stderr); err != nil {
			t.Fatal(err)
		}

		if exp, got := len(expTables), len(drv.tableMap); got != exp {
			t.Fatalf("expected enumerateTables to discover %d tables; got %d\n", exp, got)
		}

		for _, tableName := range expTables {
			if drv.tableMap[tableName] == nil {
				t.Fatalf("expected enumerateTables to discover table %q", tableName)
			}
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		rsdtAddr, tableList := genTestRDST(t, acpiRev2Plus)
		identityMapFn = func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
			return mm.Page(frame), nil
		}

		// Set bad checksum for "SSDT" and "DSDT"
		for _, header := range tableList {
			switch string(header.Signature[:]) {
			case "SSDT", dsdtSignature:
				header.Checksum++
			}
		}

		drv := &acpiDriver{
			rsdtAddr: rsdtAddr,
			useXSDT:  true,
		}

		if err := drv.enumerateTables(os.Stderr); err != nil {
			t.Fatal(err)
		}

		expTables := []string{"APIC", "FACP"}

		if exp, got := len(expTables), len(drv.tableMap); got != exp {
			t.Fatalf("expected enumerateTables to discover %d tables; got %d\n", exp, got)
		}

		for _, tableName := range expTables {
			if drv.tableMap[tableName] == nil {
				t.Fatalf("expected enumerateTables to discover table %q", tableName)
			}
		}
	})
}

func TestMapACPITableErrors(t *testing.T) {
	defer func() {
		identityMapFn = vmm.IdentityMapRegion
	}()

	var (
		callCount int
		expErr    = &kernel.Error{Module: "test", Message: "identityMapRegion failed"}
		header    table.SDTHeader
	)

	identityMapFn = func(frame mm.Frame, _ uintptr, _ vmm.PageTableEntryFlag) (mm.Page, *kernel.Error) {
		callCount++
		if callCount >= 2 {
			return 0, expErr
		}

		return mm.PageFromAddress(uintptr(unsafe.Pointer(&header))), nil
	}

	// Test errors while mapping the table contents and the table header
	for i := 0; i < 2; i++ {
		if _, _, err := mapACPITable(0xf00); err != expErr {
			t.Errorf("[spec %d]; expected to get an error\n", i)
		}
	}
}

func genTestRDST(t *testing.T, acpiVersion uint8) (rsdtAddr uintptr, tableList []*table.SDTHeader) {
	dumpFiles, err := filepath.Glob(pkgDir() + "/table/tabletest/*.aml")
	if err != nil {
		t.Fatal(err)
	}

	var fadt, dsdt *table.SDTHeader
	var dsdtIndex int

	for index, df := range dumpFiles {
		dumpData, err := ioutil.ReadFile(df)
		if err != nil {
			t.Fatal(err)
		}

		header := (*table.SDTHeader)(unsafe.Pointer(&dumpData[0]))
		tableName := string(header.Signature[:])
		switch tableName {
		case dsdtSignature, fadtSignature:
			if tableName == dsdtSignature {
				dsdt = header
				dsdtIndex = index
			} else {
				fadt = header
			}
		}

		tableList = append(tableList, header)
	}

	// Setup the pointer to the DSDT
	if fadt != nil && dsdt != nil {
		fadtHeader := (*table.FADT)(unsafe.Pointer(fadt))
		if acpiVersion == acpiRev1 {
			// Since the tests run in 64-bit mode these 32-bit addresses
			// will be invalid and cause a page fault. So we cheat and
			// encode the table index and page offset as the pointer.
			// The test code will hook identityMapFn to reconstruct the
			// correct pointer to the table contents.
			offset := vmm.PageOffset(uintptr(unsafe.Pointer(dsdt)))
			encodedTableLoc := (uintptr(dsdtIndex) << mm.PageShift) + offset
			fadtHeader.Dsdt = uint32(encodedTableLoc)
		} else {
			fadtHeader.Ext.Dsdt = uint64(uintptr(unsafe.Pointer(dsdt)))
		}
		updateChecksum(fadt)
	}

	// Assemble the RDST
	var (
		sizeofSDTHeader = unsafe.Sizeof(table.SDTHeader{})
		rsdtHeader      *table.SDTHeader
	)

	switch acpiVersion {
	case acpiRev1:
		buf := make([]byte, int(sizeofSDTHeader)+4*len(tableList))
		rsdtHeader = (*table.SDTHeader)(unsafe.Pointer(&buf[0]))
		rsdtHeader.Signature = [4]byte{'R', 'S', 'D', 'T'}
		rsdtHeader.Revision = acpiVersion
		rsdtHeader.Length = uint32(sizeofSDTHeader)

		// Since the tests run in 64-bit mode these 32-bit addresses
		// will be invalid and cause a page fault. So we cheat and
		// encode the table index and page offset as the pointer.
		// The test code will hook identityMapFn to reconstruct the
		// correct pointer to the table contents.
		for index, tableHeader := range tableList {
			offset := vmm.PageOffset(uintptr(unsafe.Pointer(tableHeader)))
			encodedTableLoc := (uintptr(index) << mm.PageShift) + offset

			*(*uint32)(unsafe.Pointer(&buf[rsdtHeader.Length])) = uint32(encodedTableLoc)
			rsdtHeader.Length += 4
		}
	default:
		buf := make([]byte, int(sizeofSDTHeader)+8*len(tableList))
		rsdtHeader = (*table.SDTHeader)(unsafe.Pointer(&buf[0]))
		rsdtHeader.Signature = [4]byte{'R', 'S', 'D', 'T'}
		rsdtHeader.Revision = acpiVersion
		rsdtHeader.Length = uint32(sizeofSDTHeader)
		for _, tableHeader := range tableList {
			// Do not include DSDT. This will be referenced via FADT
			if string(tableHeader.Signature[:]) == dsdtSignature {
				continue
			}
			*(*uint64)(unsafe.Pointer(&buf[rsdtHeader.Length])) = uint64(uintptr(unsafe.Pointer(tableHeader)))
			rsdtHeader.Length += 8
		}
	}

	updateChecksum(rsdtHeader)
	return uintptr(unsafe.Pointer(rsdtHeader)), tableList
}

func updateChecksum(header *table.SDTHeader) {
	header.Checksum = -calcChecksum(uintptr(unsafe.Pointer(header)), uintptr(header.Length))
}

func calcChecksum(tableAddr, length uintptr) uint8 {
	var checksum uint8
	for ptr := tableAddr; ptr < tableAddr+length; ptr++ {
		checksum += *(*uint8)(unsafe.Pointer(ptr))
	}

	return checksum
}

func pkgDir() string {
	_, f, _, _ := runtime.Caller(1)
	return filepath.Dir(f)
}

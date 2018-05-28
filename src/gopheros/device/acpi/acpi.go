package acpi

import (
	"gopheros/device"
	"gopheros/device/acpi/table"
	"gopheros/kernel"
	"gopheros/kernel/kfmt"
	"gopheros/kernel/mm"
	"gopheros/kernel/mm/vmm"
	"io"
	"unsafe"
)

const (
	acpiRev1     uint8 = 0
	acpiRev2Plus uint8 = 2
)

var (
	errMissingRSDP           = &kernel.Error{Module: "acpi", Message: "could not locate ACPI RSDP"}
	errTableChecksumMismatch = &kernel.Error{Module: "acpi", Message: "detected checksum mismatch while parsing ACPI table header"}

	mapFn         = vmm.Map
	identityMapFn = vmm.IdentityMapRegion
	unmapFn       = vmm.Unmap

	// RDSP must be located in the physical memory region 0xe0000 to 0xfffff
	rsdpLocationLow uintptr = 0xe0000
	rsdpLocationHi  uintptr = 0xfffff
	rsdpAlignment   uintptr = 16

	rsdpSignature = [8]byte{'R', 'S', 'D', ' ', 'P', 'T', 'R', ' '}
	fadtSignature = "FACP"
)

type acpiDriver struct {
	// rsdtAddr holds the address to the root system descriptor table.
	rsdtAddr uintptr

	// useXSDT specifies if the driver must use the XSDT or the RSDT table.
	useXSDT bool

	// The ACPI table map allows the driver to lookup an ACPI table header
	// by the table name. All tables included in this map are mapped into
	// memory.
	tableMap map[string]*table.SDTHeader
}

// DriverInit initializes this driver.
func (drv *acpiDriver) DriverInit(w io.Writer) *kernel.Error {
	if err := drv.enumerateTables(w); err != nil {
		return err
	}

	drv.printTableInfo(w)

	return nil
}

// DriverName returns the name of this driver.
func (*acpiDriver) DriverName() string {
	return "ACPI"
}

// DriverVersion returns the version of this driver.
func (*acpiDriver) DriverVersion() (uint16, uint16, uint16) {
	return 0, 0, 1
}

func (drv *acpiDriver) printTableInfo(w io.Writer) {
	for name, header := range drv.tableMap {
		kfmt.Fprintf(w, "%s at 0x%16x %6x (%6s %8s)\n",
			name,
			uintptr(unsafe.Pointer(header)),
			header.Length,
			string(header.OEMID[:]),
			string(header.OEMTableID[:]),
		)
	}
}

// enumerateTables detects and maps all ACPI tables that are present. Besides
// the table list defined by the RSDP, this method will also peek into the
// FADT (if found) looking for the address of DSDT.
func (drv *acpiDriver) enumerateTables(w io.Writer) *kernel.Error {
	header, sizeofHeader, err := mapACPITable(drv.rsdtAddr)
	if err != nil {
		return err
	}

	drv.tableMap = make(map[string]*table.SDTHeader)

	var (
		acpiRev      = header.Revision
		payloadLen   = header.Length - uint32(sizeofHeader)
		sdtAddresses []uintptr
	)

	// RSDT uses 4-byte long pointers whereas the XSDT uses 8-byte long.
	switch drv.useXSDT {
	case true:
		sdtAddresses = make([]uintptr, payloadLen>>3)
		for curPtr, i := drv.rsdtAddr+sizeofHeader, 0; i < len(sdtAddresses); curPtr, i = curPtr+8, i+1 {
			sdtAddresses[i] = uintptr(*(*uint64)(unsafe.Pointer(curPtr)))
		}
	default:
		sdtAddresses = make([]uintptr, payloadLen>>2)
		for curPtr, i := drv.rsdtAddr+sizeofHeader, 0; i < len(sdtAddresses); curPtr, i = curPtr+4, i+1 {
			sdtAddresses[i] = uintptr(*(*uint32)(unsafe.Pointer(curPtr)))
		}
	}

	for _, addr := range sdtAddresses {
		if header, _, err = mapACPITable(addr); err != nil {
			switch err {
			case errTableChecksumMismatch:
				kfmt.Fprintf(w, "%s at 0x%16x %6x [checksum mismatch; skipping]\n",
					string(header.Signature[:]),
					uintptr(unsafe.Pointer(header)),
					header.Length,
				)
				continue
			default:
				return err
			}
		}

		signature := string(header.Signature[:])
		drv.tableMap[signature] = header

		// The FADT allows us to lookup the DSDT table address
		if signature == fadtSignature {
			fadt := (*table.FADT)(unsafe.Pointer(header))

			dsdtAddr := uintptr(fadt.Dsdt)
			if acpiRev >= acpiRev2Plus {
				dsdtAddr = uintptr(fadt.Ext.Dsdt)
			}

			if header, _, err = mapACPITable(dsdtAddr); err != nil {
				switch err {
				case errTableChecksumMismatch:
					kfmt.Fprintf(w, "%s at 0x%16x %6x [checksum mismatch; skipping]\n",
						string(header.Signature[:]),
						uintptr(unsafe.Pointer(header)),
						header.Length,
					)
					continue
				default:
					return err
				}
			}

			drv.tableMap[string(header.Signature[:])] = header
		}

	}

	return nil
}

// mapACPITable attempts to map and parse the header for the ACPI table starting
// at the given address. It then uses the length field for the header to expand
// the mapping to cover the table contents and verifies the checksum before
// returning a pointer to the table header.
func mapACPITable(tableAddr uintptr) (header *table.SDTHeader, sizeofHeader uintptr, err *kernel.Error) {
	var headerPage mm.Page

	// Identity-map the table header so we can access its length field
	sizeofHeader = unsafe.Sizeof(table.SDTHeader{})
	if headerPage, err = identityMapFn(mm.FrameFromAddress(tableAddr), sizeofHeader, vmm.FlagPresent); err != nil {
		return nil, sizeofHeader, err
	}

	// Expand mapping to cover the table contents
	headerPageAddr := headerPage.Address() + vmm.PageOffset(tableAddr)
	header = (*table.SDTHeader)(unsafe.Pointer(headerPageAddr))
	if _, err = identityMapFn(mm.FrameFromAddress(tableAddr), uintptr(header.Length), vmm.FlagPresent); err != nil {
		return nil, sizeofHeader, err
	}

	if !validTable(headerPageAddr, header.Length) {
		err = errTableChecksumMismatch
	}

	return header, sizeofHeader, err
}

// locateRSDT scans the memory region [rsdpLocationLow, rsdpLocationHi] looking
// for the signature of the root system descriptor pointer (RSDP). If the RSDP
// is found and is valid, locateRSDT returns the physical address of the root
// system descriptor table (RSDT) or the extended system descriptor table (XSDT)
// if the system supports ACPI 2.0+.
func locateRSDT() (uintptr, bool, *kernel.Error) {
	var (
		rsdp  *table.RSDPDescriptor
		rsdp2 *table.ExtRSDPDescriptor
	)

	// Cleanup temporary identity mappings when the function returns
	defer func() {
		for curPage := mm.PageFromAddress(rsdpLocationLow); curPage <= mm.PageFromAddress(rsdpLocationHi); curPage++ {
			unmapFn(curPage)
		}
	}()

	// Setup temporary identity mapping so we can scan for the header
	for curPage := mm.PageFromAddress(rsdpLocationLow); curPage <= mm.PageFromAddress(rsdpLocationHi); curPage++ {
		if err := mapFn(curPage, mm.Frame(curPage), vmm.FlagPresent); err != nil {
			return 0, false, err
		}
	}

	// The RSDP should be aligned on a 16-byte boundary
checkNextBlock:
	for curPtr := rsdpLocationLow; curPtr < rsdpLocationHi; curPtr += rsdpAlignment {
		rsdp = (*table.RSDPDescriptor)(unsafe.Pointer(curPtr))
		for i, b := range rsdpSignature {
			if rsdp.Signature[i] != b {
				continue checkNextBlock
			}
		}

		if rsdp.Revision == acpiRev1 {
			if !validTable(curPtr, uint32(unsafe.Sizeof(*rsdp))) {
				continue
			}

			return uintptr(rsdp.RSDTAddr), false, nil
		}

		// System uses ACPI revision > 1 and provides an extended RSDP
		// which can be accessed at the same place.
		rsdp2 = (*table.ExtRSDPDescriptor)(unsafe.Pointer(curPtr))
		if !validTable(curPtr, uint32(unsafe.Sizeof(*rsdp2))) {
			continue
		}

		return uintptr(rsdp2.XSDTAddr), true, nil
	}

	return 0, false, errMissingRSDP
}

// validTable calculates the checksum for an ACPI table of length tableLength
// that starts at tablePtr and returns true if the table is valid.
func validTable(tablePtr uintptr, tableLength uint32) bool {
	var (
		i   uint32
		sum uint8
	)

	for i = 0; i < tableLength; i++ {
		sum += *(*uint8)(unsafe.Pointer(tablePtr + uintptr(i)))
	}

	return sum == 0
}

func probeForACPI() device.Driver {
	if rsdtAddr, useXSDT, err := locateRSDT(); err == nil {
		return &acpiDriver{
			rsdtAddr: rsdtAddr,
			useXSDT:  useXSDT,
		}
	}

	return nil
}

func init() {
	device.RegisterDriver(&device.DriverInfo{
		Order: device.DetectOrderBeforeACPI,
		Probe: probeForACPI,
	})
}

package aml

import (
	"gopheros/device/acpi/table"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

func TestParser(t *testing.T) {
	specs := [][]string{
		[]string{"DSDT.aml", "SSDT.aml"},
		[]string{"parser-testsuite-DSDT.aml"},
	}

	for specIndex, spec := range specs {
		var resolver = mockResolver{
			tableFiles: spec,
		}

		// Create default scopes
		rootNS := &scopeEntity{op: opScope, name: `\`}
		rootNS.Append(&scopeEntity{op: opScope, name: `_GPE`}) // General events in GPE register block
		rootNS.Append(&scopeEntity{op: opScope, name: `_PR_`}) // ACPI 1.0 processor namespace
		rootNS.Append(&scopeEntity{op: opScope, name: `_SB_`}) // System bus with all device objects
		rootNS.Append(&scopeEntity{op: opScope, name: `_SI_`}) // System indicators
		rootNS.Append(&scopeEntity{op: opScope, name: `_TZ_`}) // ACPI 1.0 thermal zone namespace

		p := NewParser(ioutil.Discard, rootNS)

		for _, tableName := range spec {
			tableName := strings.Replace(tableName, ".aml", "", -1)
			if err := p.ParseAML(tableName, resolver.LookupTable(tableName)); err != nil {
				t.Errorf("[spec %d] [%s]: %v", specIndex, tableName, err)
				break
			}
		}
	}
}

func pkgDir() string {
	_, f, _, _ := runtime.Caller(1)
	return filepath.Dir(f)
}

type mockResolver struct {
	tableFiles []string
}

func (m mockResolver) LookupTable(name string) *table.SDTHeader {
	pathToDumps := pkgDir() + "/../table/tabletest/"
	for _, f := range m.tableFiles {
		if !strings.Contains(f, name) {
			continue
		}

		data, err := ioutil.ReadFile(pathToDumps + f)
		if err != nil {
			panic(err)
		}

		return (*table.SDTHeader)(unsafe.Pointer(&data[0]))
	}

	return nil
}

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
			if err := p.ParseAML(0, tableName, resolver.LookupTable(tableName)); err != nil {
				t.Errorf("[spec %d] [%s]: %v", specIndex, tableName, err)
				break
			}
		}
	}
}

func TestTableHandleAssignment(t *testing.T) {
	var resolver = mockResolver{tableFiles: []string{"parser-testsuite-DSDT.aml"}}

	// Create default scopes
	rootNS := &scopeEntity{op: opScope, name: `\`}
	rootNS.Append(&scopeEntity{op: opScope, name: `_GPE`}) // General events in GPE register block
	rootNS.Append(&scopeEntity{op: opScope, name: `_PR_`}) // ACPI 1.0 processor namespace
	rootNS.Append(&scopeEntity{op: opScope, name: `_SB_`}) // System bus with all device objects
	rootNS.Append(&scopeEntity{op: opScope, name: `_SI_`}) // System indicators
	rootNS.Append(&scopeEntity{op: opScope, name: `_TZ_`}) // ACPI 1.0 thermal zone namespace

	p := NewParser(ioutil.Discard, rootNS)

	expHandle := uint8(42)
	tableName := "parser-testsuite-DSDT"
	if err := p.ParseAML(expHandle, tableName, resolver.LookupTable(tableName)); err != nil {
		t.Error(err)
	}

	// Drop all entities that were assigned the handle value
	var unloadList []Entity
	scopeVisit(0, p.root, EntityTypeAny, func(_ int, ent Entity) bool {
		if ent.TableHandle() == expHandle {
			unloadList = append(unloadList, ent)
			return false
		}
		return true
	})

	for _, ent := range unloadList {
		if p := ent.Parent(); p != nil {
			p.removeChild(ent)
		}
	}

	// We should end up with the original tree
	var visitedNodes int
	scopeVisit(0, p.root, EntityTypeAny, func(_ int, ent Entity) bool {
		visitedNodes++
		if ent.TableHandle() == expHandle {
			t.Errorf("encounted entity that should have been pruned: %#+v", ent)
		}
		return true
	})

	if exp := len(rootNS.Children()) + 1; visitedNodes != exp {
		t.Errorf("expected to visit %d nodes; visited %d", exp, visitedNodes)
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

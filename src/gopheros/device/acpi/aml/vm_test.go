package aml

import (
	"os"
	"reflect"
	"testing"
)

func TestVMInit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		resolver := &mockResolver{
			tableFiles: []string{"DSDT.aml"},
		}

		vm := NewVM(os.Stderr, resolver)
		if err := vm.Init(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		resolver := &fixedPayloadResolver{
			// invalid payload (incomplete opcode)
			payload: []byte{extOpPrefix},
		}

		expErr := &Error{message: errParsingAML.Module + ": " + errParsingAML.Error()}
		vm := NewVM(os.Stderr, resolver)
		if err := vm.Init(); !reflect.DeepEqual(err, expErr) {
			t.Fatalf("expected Init() to return errParsingAML; got %v", err)
		}
	})
}

func TestVMObjectLookups(t *testing.T) {
	resolver := &mockResolver{
		tableFiles: []string{"DSDT.aml"},
	}
	vm := NewVM(os.Stderr, resolver)
	if err := vm.Init(); err != nil {
		t.Fatal(err)
	}

	specs := []struct {
		absPath string
		match   bool
	}{
		{
			``,
			false,
		},
		{
			`\`,
			true,
		},
		{
			`\_SB_.PCI0.SBRG.PIC_`,
			true,
		},
		{
			`\_SB_.PCI0.UNKNOWN_PATH`,
			false,
		},
	}

	for specIndex, spec := range specs {
		foundMatch := vm.Lookup(spec.absPath) != nil
		if foundMatch != spec.match {
			t.Errorf("[spec %d] expected lookup match status to be %t", specIndex, spec.match)
		}
	}
}

func TestVMVisit(t *testing.T) {
	resolver := &mockResolver{
		tableFiles: []string{"parser-testsuite-DSDT.aml"},
	}
	vm := NewVM(os.Stderr, resolver)
	if err := vm.Init(); err != nil {
		t.Fatal(err)
	}

	var (
		methodCount int
		expCount    = 2
	)

	vm.Visit(EntityTypeMethod, func(_ int, ent Entity) bool {
		methodCount++
		return true
	})

	if methodCount != expCount {
		t.Fatalf("expected visitor to be invoked for %d methods; got %d", expCount, methodCount)
	}
}

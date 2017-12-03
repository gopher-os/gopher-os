package aml

import (
	"gopheros/device/acpi/table"
	"io/ioutil"
	"os"
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
		[]string{"parser-testsuite-fwd-decls-DSDT.aml"},
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

		// Inject pre-defined OSPM objects
		rootNS.Append(&constEntity{name: "_OS_", val: "gopheros"})
		rootNS.Append(&constEntity{name: "_REV", val: uint64(2)})

		p := NewParser(os.Stderr, rootNS)

		for _, tableName := range spec {
			tableName = strings.Replace(tableName, ".aml", "", -1)
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
			t.Errorf("encountered entity that should have been pruned: %#+v", ent)
		}
		return true
	})

	if exp := len(rootNS.Children()) + 1; visitedNodes != exp {
		t.Errorf("expected to visit %d nodes; visited %d", exp, visitedNodes)
	}
}

func TestParserForwardDeclParsing(t *testing.T) {
	var resolver = mockResolver{
		tableFiles: []string{"parser-testsuite-fwd-decls-DSDT.aml"},
	}

	// Create default scopes
	rootNS := &scopeEntity{op: opScope, name: `\`}
	rootNS.Append(&scopeEntity{op: opScope, name: `_GPE`}) // General events in GPE register block
	rootNS.Append(&scopeEntity{op: opScope, name: `_PR_`}) // ACPI 1.0 processor namespace
	rootNS.Append(&scopeEntity{op: opScope, name: `_SB_`}) // System bus with all device objects
	rootNS.Append(&scopeEntity{op: opScope, name: `_SI_`}) // System indicators
	rootNS.Append(&scopeEntity{op: opScope, name: `_TZ_`}) // ACPI 1.0 thermal zone namespace

	p := NewParser(ioutil.Discard, rootNS)

	for _, tableName := range resolver.tableFiles {
		tableName = strings.Replace(tableName, ".aml", "", -1)
		if err := p.ParseAML(0, tableName, resolver.LookupTable(tableName)); err != nil {
			t.Errorf("[%s]: %v", tableName, err)
			break
		}
	}
}

func TestParsePkgLength(t *testing.T) {
	specs := []struct {
		payload []byte
		exp     uint32
	}{
		// lead byte bits (6:7) indicate 1 extra byte for the len. The
		// parsed length will use bits 0:3 from the lead byte plus
		// the full 8 bits of the following byte.
		{
			[]byte{1<<6 | 7, 255},
			4087,
		},
		// lead byte bits (6:7) indicate 2 extra bytes for the len. The
		// parsed length will use bits 0:3 from the lead byte plus
		// the full 8 bits of the following bytes.
		{
			[]byte{2<<6 | 8, 255, 128},
			528376,
		},
		// lead byte bits (6:7) indicate 3 extra bytes for the len. The
		// parsed length will use bits 0:3 from the lead byte plus
		// the full 8 bits of the following bytes.
		{
			[]byte{3<<6 | 6, 255, 128, 42},
			44568566,
		},
	}

	p := &Parser{errWriter: ioutil.Discard}

	for specIndex, spec := range specs {
		mockParserPayload(p, spec.payload)
		got, ok := p.parsePkgLength()
		if !ok {
			t.Errorf("[spec %d] parsePkgLength returned false", specIndex)
			continue
		}

		if got != spec.exp {
			t.Errorf("[spec %d] expected parsePkgLength to return %d; got %d", specIndex, spec.exp, got)
		}
	}
}

func TestParserErrorHandling(t *testing.T) {
	p := &Parser{
		errWriter: ioutil.Discard,
	}

	t.Run("ParseAML errors", func(t *testing.T) {
		t.Run("parseObjList error", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			// Setup resolver to serve an AML stream containing an invalid opcode
			header := mockParserPayload(p, []byte{0x5b, 0x00})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected ParseAML to return an error")
			}

			// Setup resolver to serve an AML stream containing an incomplete extended opcode
			header = mockParserPayload(p, []byte{0x5b})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected ParseAML to return an error")
			}
		})

		t.Run("unresolved entities", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			// Inject a reference entity to the tree
			p.root.Append(&namedReference{
				targetName: "UNKNOWN",
			})

			// Setup resolver to serve an empty AML stream
			header := mockParserPayload(p, nil)

			if err := p.ParseAML(uint8(42), "DSDT", header); err != errResolvingEntities {
				t.Fatalf("expected ParseAML to return errResolvingEntities; got %v", err)
			}
		})
	})

	t.Run("parseObj errors", func(t *testing.T) {
		t.Run("parsePkgLength error", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			// Setup resolver to serve an AML stream containing an incomplete
			// buffer specification
			header := mockParserPayload(p, []byte{byte(opBuffer)})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parsePkgLength to return an error")
			}
		})

		t.Run("incomplete object list", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			// Setup resolver to serve an AML stream containing an incomplete
			// buffer arglist specification
			header := mockParserPayload(p, []byte{byte(opBuffer), 0x10})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parsePkgLength to return an error")
			}
		})
	})

	t.Run("finalizeObj errors", func(t *testing.T) {
		t.Run("else without matching if", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}
			p.root.Append(&constEntity{val: 0x42})
			p.root.Append(&scopeEntity{op: opElse})

			// Setup resolver to serve an AML stream containing an
			// empty else statement without a matching if
			header := mockParserPayload(p, []byte{byte(opElse), 0x0})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected finalizeObj to return an error")
			}
		})

	})

	t.Run("parseScope errors", func(t *testing.T) {
		t.Run("parseNameString error", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			header := mockParserPayload(p, []byte{
				byte(opScope),
				0x10, // pkglen
			})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parseScope to return an error")
			}
		})

		t.Run("unknown scope", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			header := mockParserPayload(p, []byte{
				byte(opScope),
				0x10, // pkglen
				'F', 'O', 'O', 'F',
			})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parseScope to return an error")
			}
		})

		t.Run("nameless scope", func(t *testing.T) {
			p.root = &scopeEntity{}

			header := mockParserPayload(p, []byte{
				byte(opScope),
				0x02, // pkglen
				'\\', // scope name: "\" (root scope)
				0x00, // null string
			})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parseScope to return an error")
			}
		})
	})

	t.Run("parseNamespacedObj errors", func(t *testing.T) {
		t.Run("parseNameString error", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			mockParserPayload(p, nil)

			if p.parseNamespacedObj(opDevice, 10) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})

		t.Run("scope lookup error", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			header := mockParserPayload(p, []byte{'^', 'F', 'A', 'B', 'C'})

			p.scopeEnter(p.root)
			if p.parseNamespacedObj(opDevice, header.Length) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})

		t.Run("error parsing method arg count", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}

			header := mockParserPayload(p, []byte{'F', 'A', 'B', 'C'})

			p.scopeEnter(p.root)
			if p.parseNamespacedObj(opMethod, header.Length) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})
	})

	t.Run("parseArg bytelist errors", func(t *testing.T) {
		p.root = &scopeEntity{op: opScope, name: `\`}

		mockParserPayload(p, nil)

		if p.parseArg(new(opcodeInfo), new(unnamedEntity), 0, opArgByteList, 42) {
			t.Fatal("expected parseNamespacedObj to return false")
		}
	})

	t.Run("parseNamedRef errors", func(t *testing.T) {
		t.Run("missing args", func(t *testing.T) {
			p.root = &scopeEntity{op: opScope, name: `\`}
			p.methodArgCount = map[string]uint8{
				"MTHD": 10,
			}

			mockParserPayload(p, []byte{
				'M', 'T', 'H', 'D',
				byte(opIf), // Incomplete type2 opcode
			})

			p.scopeEnter(p.root)
			if p.parseNamedRef() {
				t.Fatal("expected parseNamedRef to return false")
			}
		})
	})

	t.Run("parseFieldList errors", func(t *testing.T) {
		specs := []struct {
			op            opcode
			args          []interface{}
			maxReadOffset uint32
			payload       []byte
		}{
			// Invalid arg count for opField
			{
				opField,
				nil,
				0,
				nil,
			},
			// Wrong arg type for opField
			{
				opField,
				[]interface{}{0, uint64(42)},
				0,
				nil,
			},
			{
				opField,
				[]interface{}{"FLD0", uint32(42)},
				0,
				nil,
			},
			// Invalid arg count for opIndexField
			{
				opIndexField,
				nil,
				0,
				nil,
			},
			// Wrong arg type for opIndexField
			{
				opIndexField,
				[]interface{}{0, "FLD1", "FLD2"},
				0,
				nil,
			},
			{
				opIndexField,
				[]interface{}{"FLD0", 0, "FLD2"},
				0,
				nil,
			},
			{
				opIndexField,
				[]interface{}{"FLD0", "FLD1", 0},
				0,
				nil,
			},
			// unexpected EOF parsing fields
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				nil,
			},
			// reserved field (0x00) with missing pkgLen
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x00},
			},
			// access field (0x01) with missing accessType
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x01},
			},
			// access field (0x01) with missing attribute byte
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x01, 0x01},
			},
			// connect field (0x02) with incomplete TermObject => Buffer arg
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x02, byte(opBuffer)},
			},
			// extended access field (0x03) with missing ext. accessType
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x03},
			},
			// extended access field (0x03) with missing ext. attribute byte
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x03, 0x01},
			},
			// extended access field (0x03) with missing access byte count value
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x03, 0x01, 0x02},
			},
			// named field with invalid name
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0xff},
			},
			// named field with invalid pkgLen
			{
				opField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{'N', 'A', 'M', 'E'},
			},
		}

		for specIndex, spec := range specs {
			mockParserPayload(p, spec.payload)

			if p.parseFieldList(spec.op, spec.args, spec.maxReadOffset) {
				t.Errorf("[spec %d] expected parseFieldLis to return false", specIndex)
			}
		}
	})

	t.Run("parsePkgLength errors", func(t *testing.T) {
		specs := [][]byte{
			// lead byte bits (6:7) indicate 1 extra byte that is missing
			[]byte{1 << 6},
			// lead byte bits (6:7) indicate 2 extra bytes with the 1st and then 2nd missing
			[]byte{2 << 6},
			[]byte{2 << 6, 0x1},
			// lead byte bits (6:7) indicate 3 extra bytes with the 1st and then 2nd and then 3rd missing
			[]byte{3 << 6},
			[]byte{3 << 6, 0x1},
			[]byte{3 << 6, 0x1, 0x2},
		}

		for specIndex, spec := range specs {
			mockParserPayload(p, spec)

			if _, ok := p.parsePkgLength(); ok {
				t.Errorf("[spec %d] expected parsePkgLength to return false", specIndex)
			}
		}
	})

	t.Run("parseString errors", func(t *testing.T) {
		specs := [][]byte{
			// Unexpected EOF before terminating null byte
			[]byte{'A'},
			// Characters outside the allowed [0x01, 0x7f] range
			[]byte{'A', 0xba, 0xdf, 0x00},
		}

		for specIndex, spec := range specs {
			mockParserPayload(p, spec)

			if _, ok := p.parseString(); ok {
				t.Errorf("[spec %d] expected parseString to return false", specIndex)
			}
		}
	})

	t.Run("parseTarget errors", func(t *testing.T) {
		t.Run("unexpected opcode", func(t *testing.T) {
			// Unexpected opcode
			mockParserPayload(p, []byte{byte(opAnd)})

			if _, ok := p.parseTarget(); ok {
				t.Error("expected parseTarget to return false")
			}
		})

		t.Run("corrupted data", func(t *testing.T) {
			// Invalid opcode and not a method invocation nor a namestring
			mockParserPayload(p, []byte{0xba, 0xad})

			if _, ok := p.parseTarget(); ok {
				t.Error("expected parseTarget to return false")
			}
		})
	})

	t.Run("parseNameString errors", func(t *testing.T) {
		t.Run("EOF while parsing path prefix", func(t *testing.T) {
			mockParserPayload(p, []byte{'^'})

			if _, ok := p.parseNameString(); ok {
				t.Error("expected parseNameString to return false")
			}
		})

		t.Run("EOF while parsing multiname path", func(t *testing.T) {
			specs := [][]byte{
				// multiname path prefix but no data following
				[]byte{0x2f},
				[]byte{
					0x2f, // multiname path prefix
					0x0,  // no segments (segments must be > 0)
				},
				[]byte{
					0x2f, // multiname path prefix
					0x1,  // 1 expected segment but no more data available
				},
				[]byte{
					'\\', // RootChar and no more data
				},
			}

			for specIndex, spec := range specs {
				mockParserPayload(p, spec)
				if _, ok := p.parseNameString(); ok {
					t.Errorf("[spec %d] expected parseNameString to return false", specIndex)
				}
			}
		})
	})
}

func TestDetectMethodDeclarations(t *testing.T) {
	p := &Parser{
		errWriter: ioutil.Discard,
	}

	validMethod := []byte{
		byte(opMethod),
		5, // pkgLen
		'M', 'T', 'H', 'D',
		2, // flags (2 args)
	}

	t.Run("success", func(t *testing.T) {
		mockParserPayload(p, validMethod)
		p.methodArgCount = make(map[string]uint8)
		p.detectMethodDeclarations()

		argCount, inMap := p.methodArgCount["MTHD"]
		if !inMap {
			t.Error(`detectMethodDeclarations failed to parse method "MTHD"`)
		}

		if exp := uint8(2); argCount != exp {
			t.Errorf(`expected arg count for "MTHD" to be %d; got %d`, exp, argCount)
		}
	})

	t.Run("bad pkgLen", func(t *testing.T) {
		mockParserPayload(p, []byte{
			byte(opMethod),
			// lead byte bits (6:7) indicate 1 extra byte that is missing
			byte(1 << 6),
		})

		p.methodArgCount = make(map[string]uint8)
		p.detectMethodDeclarations()
	})

	t.Run("error parsing namestring", func(t *testing.T) {
		mockParserPayload(p, append([]byte{
			byte(opMethod),
			byte(5), // pkgLen
			10,      // bogus char, not part of namestring
		}, validMethod...))

		p.methodArgCount = make(map[string]uint8)
		p.detectMethodDeclarations()

		argCount, inMap := p.methodArgCount["MTHD"]
		if !inMap {
			t.Error(`detectMethodDeclarations failed to parse method "MTHD"`)
		}

		if exp := uint8(2); argCount != exp {
			t.Errorf(`expected arg count for "MTHD" to be %d; got %d`, exp, argCount)
		}
	})

	t.Run("error parsing method flags", func(t *testing.T) {
		mockParserPayload(p, []byte{
			byte(opMethod),
			byte(5), // pkgLen
			'F', 'O', 'O', 'F',
			// Missing flag byte
		})

		p.methodArgCount = make(map[string]uint8)
		p.detectMethodDeclarations()
	})
}

func mockParserPayload(p *Parser, payload []byte) *table.SDTHeader {
	resolver := fixedPayloadResolver{payload}
	header := resolver.LookupTable("DSDT")
	p.r.Init(
		uintptr(unsafe.Pointer(header)),
		header.Length,
		uint32(unsafe.Sizeof(table.SDTHeader{})),
	)

	return resolver.LookupTable("DSDT")
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

type fixedPayloadResolver struct {
	payload []byte
}

func (f fixedPayloadResolver) LookupTable(name string) *table.SDTHeader {
	hdrLen := int(unsafe.Sizeof(table.SDTHeader{}))
	buf := make([]byte, len(f.payload)+hdrLen)
	copy(buf[hdrLen:], f.payload)

	hdr := (*table.SDTHeader)(unsafe.Pointer(&buf[0]))
	hdr.Length = uint32(len(buf))

	return hdr
}

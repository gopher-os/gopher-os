package parser

import (
	"gopheros/device/acpi/aml/entity"
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

		p := NewParser(os.Stderr, genDefaultScopes())

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

	rootNS := genDefaultScopes()
	p := NewParser(ioutil.Discard, rootNS)

	expHandle := uint8(0x0f)
	tableName := "parser-testsuite-DSDT"
	if err := p.ParseAML(expHandle, tableName, resolver.LookupTable(tableName)); err != nil {
		t.Error(err)
	}

	// Drop all entities that were assigned the handle value
	var unloadList []entity.Entity
	entity.Visit(0, p.root, entity.TypeAny, func(_ int, ent entity.Entity) bool {
		if ent.TableHandle() == expHandle {
			unloadList = append(unloadList, ent)
			return false
		}
		return true
	})

	for _, ent := range unloadList {
		if p := ent.Parent(); p != nil {
			p.Remove(ent)
		}
	}

	// We should end up with the original tree
	var visitedNodes int
	entity.Visit(0, p.root, entity.TypeAny, func(_ int, ent entity.Entity) bool {
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

	p := NewParser(ioutil.Discard, genDefaultScopes())

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
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

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
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			// Inject a reference entity to the tree
			p.root.Append(entity.NewReference(42, "UNKNOWN"))

			// Setup resolver to serve an empty AML stream
			header := mockParserPayload(p, nil)

			if err := p.ParseAML(uint8(42), "DSDT", header); err != errResolvingEntities {
				t.Fatalf("expected ParseAML to return errResolvingEntities; got %v", err)
			}
		})
	})

	t.Run("parseObj errors", func(t *testing.T) {
		t.Run("parsePkgLength error", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			// Setup resolver to serve an AML stream containing an incomplete
			// buffer specification
			header := mockParserPayload(p, []byte{byte(entity.OpBuffer)})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parsePkgLength to return an error")
			}
		})

		t.Run("incomplete object list", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			// Setup resolver to serve an AML stream containing an incomplete
			// buffer arglist specification
			header := mockParserPayload(p, []byte{byte(entity.OpBuffer), 0x10})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parsePkgLength to return an error")
			}
		})
	})

	t.Run("finalizeObj errors", func(t *testing.T) {
		t.Run("else without matching if", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)
			p.root.Append(entity.NewConst(entity.OpDwordPrefix, 42, uint64(0x42)))

			// Setup resolver to serve an AML stream containing an
			// empty else statement without a matching if
			header := mockParserPayload(p, []byte{byte(entity.OpElse), 0x0})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected finalizeObj to return an error")
			}
		})

	})

	t.Run("parseScope errors", func(t *testing.T) {
		t.Run("parseNameString error", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			header := mockParserPayload(p, []byte{
				byte(entity.OpScope),
				0x10, // pkglen
			})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parseScope to return an error")
			}
		})

		t.Run("unknown scope", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			header := mockParserPayload(p, []byte{
				byte(entity.OpScope),
				0x10, // pkglen
				'F', 'O', 'O', 'F',
			})

			if err := p.ParseAML(uint8(42), "DSDT", header); err == nil {
				t.Fatal("expected parseScope to return an error")
			}
		})

		t.Run("nameless scope", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, ``)

			header := mockParserPayload(p, []byte{
				byte(entity.OpScope),
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
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			mockParserPayload(p, nil)

			devInfo := &opcodeTable[0x6a]
			if p.parseNamespacedObj(devInfo, 10) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})

		t.Run("scope lookup error", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			header := mockParserPayload(p, []byte{'^', 'F', 'A', 'B', 'C'})

			p.scopeEnter(p.root)
			devInfo := &opcodeTable[0x6a]
			if p.parseNamespacedObj(devInfo, header.Length) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})

		t.Run("unsupported namespaced entity", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			header := mockParserPayload(p, []byte{'F', 'A', 'B', 'C'})

			p.scopeEnter(p.root)

			// We just pass a random non-namespaced opcode table entry to parseNamespacedObj
			zeroInfo := &opcodeTable[0x00]
			if p.parseNamespacedObj(zeroInfo, header.Length) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})

		t.Run("error parsing args after name", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)

			header := mockParserPayload(p, []byte{'F', 'A', 'B', 'C'})

			p.scopeEnter(p.root)
			methodInfo := &opcodeTable[0x0d]
			if p.parseNamespacedObj(methodInfo, header.Length) {
				t.Fatal("expected parseNamespacedObj to return false")
			}
		})
	})

	t.Run("parseArg bytelist errors", func(t *testing.T) {
		p.root = entity.NewScope(entity.OpScope, 42, `\`)

		mockParserPayload(p, nil)

		if p.parseArg(new(opcodeInfo), entity.NewGeneric(0, 0), 0, opArgByteList, 42) {
			t.Fatal("expected parseNamespacedObj to return false")
		}
	})

	t.Run("parseNamedRef errors", func(t *testing.T) {
		t.Run("missing args", func(t *testing.T) {
			p.root = entity.NewScope(entity.OpScope, 42, `\`)
			p.methodArgCount = map[string]uint8{
				"MTHD": 10,
			}

			mockParserPayload(p, []byte{
				'M', 'T', 'H', 'D',
				byte(entity.OpIf), // Incomplete type2 opcode
			})

			p.scopeEnter(p.root)
			if p.parseNamedRef() {
				t.Fatal("expected parseNamedRef to return false")
			}
		})
	})

	t.Run("parseFieldList errors", func(t *testing.T) {
		specs := []struct {
			op            entity.AMLOpcode
			args          []interface{}
			maxReadOffset uint32
			payload       []byte
		}{
			// Invalid arg count for entity.OpField
			{
				entity.OpField,
				nil,
				0,
				nil,
			},
			// Wrong arg type for entity.OpField
			{
				entity.OpField,
				[]interface{}{0, uint64(42)},
				0,
				nil,
			},
			{
				entity.OpField,
				[]interface{}{"FLD0", uint32(42)},
				0,
				nil,
			},
			// Invalid arg count for entity.OpIndexField
			{
				entity.OpIndexField,
				nil,
				0,
				nil,
			},
			// Wrong arg type for entity.OpIndexField
			{
				entity.OpIndexField,
				[]interface{}{0, "FLD1", "FLD2"},
				0,
				nil,
			},
			{
				entity.OpIndexField,
				[]interface{}{"FLD0", 0, "FLD2"},
				0,
				nil,
			},
			{
				entity.OpIndexField,
				[]interface{}{"FLD0", "FLD1", 0},
				0,
				nil,
			},
			// Invalid arg count for entity.OpBankField
			{
				entity.OpBankField,
				nil,
				0,
				nil,
			},
			// Wrong arg type for entity.OpBankField
			{
				entity.OpBankField,
				[]interface{}{0, "FLD1", "FLD2"},
				0,
				nil,
			},
			{
				entity.OpBankField,
				[]interface{}{"FLD0", 0, "FLD2"},
				0,
				nil,
			},
			{
				entity.OpBankField,
				[]interface{}{"FLD0", "FLD1", 0},
				0,
				nil,
			},
			// unexpected EOF parsing fields
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				nil,
			},
			// reserved field (0x00) with missing pkgLen
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x00},
			},
			// access field (0x01) with missing accessType
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x01},
			},
			// access field (0x01) with missing attribute byte
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x01, 0x01},
			},
			// connect field (0x02) with incomplete TermObject => Buffer arg
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x02, byte(entity.OpBuffer)},
			},
			// extended access field (0x03) with missing ext. accessType
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x03},
			},
			// extended access field (0x03) with missing ext. attribute byte
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x03, 0x01},
			},
			// extended access field (0x03) with missing access byte count value
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0x03, 0x01, 0x02},
			},
			// named field with invalid name
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{0xff},
			},
			// named field with invalid pkgLen
			{
				entity.OpField,
				[]interface{}{"FLD0", uint64(42)},
				128,
				[]byte{'N', 'A', 'M', 'E'},
			},
		}

		for specIndex, spec := range specs {
			mockParserPayload(p, spec.payload)

			if p.parseFieldList(entity.NewField(42), spec.maxReadOffset) {
				t.Errorf("[spec %d] expected parseFieldLis to return false", specIndex)
			}
		}

		t.Run("non-field entity argument", func(t *testing.T) {
			if p.parseFieldList(entity.NewDevice(42, "DEV0"), 128) {
				t.Fatal("expected parseFieldList to return false when a non-field argument is passed to it")
			}
		})
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
			mockParserPayload(p, []byte{byte(entity.OpAnd)})

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
		byte(entity.OpMethod),
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
			byte(entity.OpMethod),
			// lead byte bits (6:7) indicate 1 extra byte that is missing
			byte(1 << 6),
		})

		p.methodArgCount = make(map[string]uint8)
		p.detectMethodDeclarations()
	})

	t.Run("error parsing namestring", func(t *testing.T) {
		mockParserPayload(p, append([]byte{
			byte(entity.OpMethod),
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
			byte(entity.OpMethod),
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
	pathToDumps := pkgDir() + "/../../table/tabletest/"
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

func genDefaultScopes() entity.Container {
	rootNS := entity.NewScope(entity.OpScope, 42, `\`)
	rootNS.Append(entity.NewScope(entity.OpScope, 42, `_GPE`)) // General events in GPE register block
	rootNS.Append(entity.NewScope(entity.OpScope, 42, `_PR_`)) // ACPI 1.0 processor namespace
	rootNS.Append(entity.NewScope(entity.OpScope, 42, `_SB_`)) // System bus with all device objects
	rootNS.Append(entity.NewScope(entity.OpScope, 42, `_SI_`)) // System indicators
	rootNS.Append(entity.NewScope(entity.OpScope, 42, `_TZ_`)) // ACPI 1.0 thermal zone namespace

	// Inject pre-defined OSPM objects
	rootNS.Append(namedConst(entity.NewConst(entity.OpStringPrefix, 42, "gopheros"), "_OS_"))
	rootNS.Append(namedConst(entity.NewConst(entity.OpStringPrefix, 42, uint64(2)), "_REV"))

	return rootNS
}

func namedConst(ent *entity.Const, name string) *entity.Const {
	ent.SetName(name)
	return ent
}

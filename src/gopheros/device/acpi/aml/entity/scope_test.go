package entity

import (
	"reflect"
	"testing"
)

func TestResolveScopedPath(t *testing.T) {
	scopeMap := genTestScopes()

	specs := []struct {
		curScope   Container
		pathExpr   string
		wantParent Entity
		wantName   string
	}{
		{
			scopeMap["IDE0"].(Container),
			`\_SB_`,
			scopeMap[`\`],
			"_SB_",
		},
		{
			scopeMap["IDE0"].(Container),
			`^FOO`,
			scopeMap[`PCI0`],
			"FOO",
		},
		{
			scopeMap["IDE0"].(Container),
			`^^FOO`,
			scopeMap[`_SB_`],
			"FOO",
		},
		{
			scopeMap["IDE0"].(Container),
			`_ADR`,
			scopeMap[`IDE0`],
			"_ADR",
		},
		// Paths with dots
		{
			scopeMap["IDE0"].(Container),
			`\_SB_.PCI0.IDE0._ADR`,
			scopeMap[`IDE0`],
			"_ADR",
		},
		{
			scopeMap["PCI0"].(Container),
			`IDE0._ADR`,
			scopeMap[`IDE0`],
			"_ADR",
		},
		{
			scopeMap["PCI0"].(Container),
			`_CRS`,
			scopeMap[`PCI0`],
			"_CRS",
		},
		// Bad queries
		{
			scopeMap["PCI0"].(Container),
			`FOO.BAR.BAZ`,
			nil,
			"",
		},
		{
			scopeMap["PCI0"].(Container),
			``,
			nil,
			"",
		},
		{
			scopeMap["PCI0"].(Container),
			`\`,
			nil,
			"",
		},
		{
			scopeMap["PCI0"].(Container),
			`^^^^^^^^^BADPATH`,
			nil,
			"",
		},
	}

	root := scopeMap[`\`].(Container)
	for specIndex, spec := range specs {
		gotParent, gotName := ResolveScopedPath(spec.curScope, root, spec.pathExpr)
		if !reflect.DeepEqual(gotParent, spec.wantParent) {
			t.Errorf("[spec %d] expected lookup to return %#v; got %#v", specIndex, spec.wantParent, gotParent)
			continue
		}

		if gotName != spec.wantName {
			t.Errorf("[spec %d] expected lookup to return node name %q; got %q", specIndex, spec.wantName, gotName)
		}
	}
}

func TestFindInScope(t *testing.T) {
	scopeMap := genTestScopes()

	specs := []struct {
		curScope Container
		lookup   string
		want     Entity
	}{
		// Search rules do not apply for these cases
		{
			scopeMap["PCI0"].(Container),
			`\`,
			scopeMap[`\`],
		},
		{
			scopeMap["PCI0"].(Container),
			"IDE0._ADR",
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"].(Container),
			"^^PCI0.IDE0._ADR",
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"].(Container),
			`\_SB_.PCI0.IDE0._ADR`,
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"].(Container),
			`\_SB_.PCI0`,
			scopeMap["PCI0"],
		},
		{
			scopeMap["IDE0"].(Container),
			`^`,
			scopeMap["PCI0"],
		},
		// Bad queries
		{
			scopeMap["_SB_"].(Container),
			"PCI0.USB._CRS",
			nil,
		},
		{
			scopeMap["IDE0"].(Container),
			"^^^^^^^^^^^^^^^^^^^",
			nil,
		},
		{
			scopeMap["IDE0"].(Container),
			`^^^^^^^^^^^FOO`,
			nil,
		},
		{
			scopeMap["IDE0"].(Container),
			"FOO",
			nil,
		},
		{
			scopeMap["IDE0"].(Container),
			"",
			nil,
		},
		// Search rules apply for these cases
		{
			scopeMap["IDE0"].(Container),
			"_CRS",
			scopeMap["_CRS"],
		},
	}

	root := scopeMap[`\`].(Container)
	for specIndex, spec := range specs {
		if got := FindInScope(spec.curScope, root, spec.lookup); !reflect.DeepEqual(got, spec.want) {
			t.Errorf("[spec %d] expected lookup to return %#v; got %#v", specIndex, spec.want, got)
		}
	}
}

func genTestScopes() map[string]Entity {
	// Setup the example tree from page 252 of the acpi 6.2 spec
	// \
	//  SB
	//    \
	//     PCI0
	//         | _CRS
	//         \
	//          IDE0
	//              | _ADR
	ideScope := NewScope(OpScope, 42, `IDE0`)
	pciScope := NewScope(OpScope, 42, `PCI0`)
	sbScope := NewScope(OpScope, 42, `_SB_`)
	rootScope := NewScope(OpScope, 42, `\`)

	adr := NewMethod(42, `_ADR`)
	crs := NewMethod(42, `_CRS`)

	// Setup tree
	ideScope.Append(adr)
	pciScope.Append(crs)
	pciScope.Append(ideScope)
	sbScope.Append(pciScope)
	rootScope.Append(sbScope)

	return map[string]Entity{
		"IDE0": ideScope,
		"PCI0": pciScope,
		"_SB_": sbScope,
		"\\":   rootScope,
		"_ADR": adr,
		"_CRS": crs,
	}
}

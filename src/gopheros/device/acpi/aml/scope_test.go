package aml

import (
	"reflect"
	"testing"
)

func TestScopeVisit(t *testing.T) {
	scopeMap := genTestScopes()
	root := scopeMap[`\`].(*scopeEntity)

	// Append special entities under IDE0
	ide := scopeMap["IDE0"].(*scopeEntity)
	ide.Append(&Device{})
	ide.Append(&namedEntity{op: opProcessor})
	ide.Append(&namedEntity{op: opProcessor})
	ide.Append(&namedEntity{op: opPowerRes})
	ide.Append(&namedEntity{op: opPowerRes})
	ide.Append(&namedEntity{op: opPowerRes})
	ide.Append(&namedEntity{op: opThermalZone})
	ide.Append(&namedEntity{op: opThermalZone})
	ide.Append(&namedEntity{op: opThermalZone})
	ide.Append(&namedEntity{op: opThermalZone})
	ide.Append(&Method{})
	ide.Append(&Method{})
	ide.Append(&Method{})
	ide.Append(&Method{})
	ide.Append(&Method{})

	specs := []struct {
		searchType    EntityType
		keepRecursing bool
		wantHits      int
	}{
		{EntityTypeAny, true, 21},
		{EntityTypeAny, false, 1},
		{EntityTypeDevice, true, 1},
		{EntityTypeProcessor, true, 2},
		{EntityTypePowerResource, true, 3},
		{EntityTypeThermalZone, true, 4},
		{EntityTypeMethod, true, 5},
	}

	for specIndex, spec := range specs {
		var hits int
		scopeVisit(0, root, spec.searchType, func(_ int, obj Entity) bool {
			hits++
			return spec.keepRecursing
		})

		if hits != spec.wantHits {
			t.Errorf("[spec %d] expected visitor to be called %d times; got %d", specIndex, spec.wantHits, hits)
		}
	}
}

func TestScopeResolvePath(t *testing.T) {
	scopeMap := genTestScopes()

	specs := []struct {
		curScope   ScopeEntity
		pathExpr   string
		wantParent Entity
		wantName   string
	}{
		{
			scopeMap["IDE0"].(*scopeEntity),
			`\_SB_`,
			scopeMap[`\`],
			"_SB_",
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`^FOO`,
			scopeMap[`PCI0`],
			"FOO",
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`^^FOO`,
			scopeMap[`_SB_`],
			"FOO",
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`_ADR`,
			scopeMap[`IDE0`],
			"_ADR",
		},
		// Paths with dots
		{
			scopeMap["IDE0"].(*scopeEntity),
			`\_SB_.PCI0.IDE0._ADR`,
			scopeMap[`IDE0`],
			"_ADR",
		},
		{
			scopeMap["PCI0"].(*scopeEntity),
			`IDE0._ADR`,
			scopeMap[`IDE0`],
			"_ADR",
		},
		{
			scopeMap["PCI0"].(*scopeEntity),
			`_CRS`,
			scopeMap[`PCI0`],
			"_CRS",
		},
		// Bad queries
		{
			scopeMap["PCI0"].(*scopeEntity),
			`FOO.BAR.BAZ`,
			nil,
			"",
		},
		{
			scopeMap["PCI0"].(*scopeEntity),
			``,
			nil,
			"",
		},
		{
			scopeMap["PCI0"].(*scopeEntity),
			`\`,
			nil,
			"",
		},
		{
			scopeMap["PCI0"].(*scopeEntity),
			`^^^^^^^^^BADPATH`,
			nil,
			"",
		},
	}

	root := scopeMap[`\`].(*scopeEntity)
	for specIndex, spec := range specs {
		gotParent, gotName := scopeResolvePath(spec.curScope, root, spec.pathExpr)
		if !reflect.DeepEqual(gotParent, spec.wantParent) {
			t.Errorf("[spec %d] expected lookup to return %#v; got %#v", specIndex, spec.wantParent, gotParent)
			continue
		}

		if gotName != spec.wantName {
			t.Errorf("[spec %d] expected lookup to return node name %q; got %q", specIndex, spec.wantName, gotName)
		}
	}
}

func TestScopeFind(t *testing.T) {
	scopeMap := genTestScopes()

	specs := []struct {
		curScope ScopeEntity
		lookup   string
		want     Entity
	}{
		// Search rules do not apply for these cases
		{
			scopeMap["PCI0"].(*scopeEntity),
			`\`,
			scopeMap[`\`],
		},
		{
			scopeMap["PCI0"].(*scopeEntity),
			"IDE0._ADR",
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			"^^PCI0.IDE0._ADR",
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`\_SB_.PCI0.IDE0._ADR`,
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`\_SB_.PCI0`,
			scopeMap["PCI0"],
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`^`,
			scopeMap["PCI0"],
		},
		// Bad queries
		{
			scopeMap["_SB_"].(*scopeEntity),
			"PCI0.USB._CRS",
			nil,
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			"^^^^^^^^^^^^^^^^^^^",
			nil,
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			`^^^^^^^^^^^FOO`,
			nil,
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			"FOO",
			nil,
		},
		{
			scopeMap["IDE0"].(*scopeEntity),
			"",
			nil,
		},
		// Search rules apply for these cases
		{
			scopeMap["IDE0"].(*scopeEntity),
			"_CRS",
			scopeMap["_CRS"],
		},
	}

	root := scopeMap[`\`].(*scopeEntity)
	for specIndex, spec := range specs {
		if got := scopeFind(spec.curScope, root, spec.lookup); !reflect.DeepEqual(got, spec.want) {
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
	ideScope := &scopeEntity{name: `IDE0`}
	pciScope := &scopeEntity{name: `PCI0`}
	sbScope := &scopeEntity{name: `_SB_`}
	rootScope := &scopeEntity{name: `\`}

	adr := &namedEntity{name: `_ADR`}
	crs := &namedEntity{name: `_CRS`}

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

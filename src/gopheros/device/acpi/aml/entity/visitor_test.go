package entity

import "testing"

func TestScopeVisit(t *testing.T) {
	tableHandle := uint8(42)
	keepRecursing := func(Entity) bool { return true }
	stopRecursing := func(Entity) bool { return false }

	// Append special entities under IDE0
	root := NewScope(tableHandle, "IDE0")
	root.Append(NewDevice(tableHandle, "DEV0"))
	root.Append(NewProcessor(tableHandle, "FOO0"))
	root.Append(NewProcessor(tableHandle, "FOO0"))
	root.Append(NewPowerResource(tableHandle, "FOO0"))
	root.Append(NewPowerResource(tableHandle, "FOO0"))
	root.Append(NewPowerResource(tableHandle, "FOO0"))
	root.Append(NewThermalZone(tableHandle, "FOO0"))
	root.Append(NewThermalZone(tableHandle, "FOO0"))
	root.Append(NewThermalZone(tableHandle, "FOO0"))
	root.Append(NewThermalZone(tableHandle, "FOO0"))
	root.Append(NewMethod(tableHandle, "MTH0"))
	root.Append(NewMethod(tableHandle, "MTH1"))
	root.Append(NewMethod(tableHandle, "MTH2"))
	root.Append(NewMethod(tableHandle, "MTH3"))
	root.Append(NewMethod(tableHandle, "MTH4"))
	root.Append(NewMutex(tableHandle))
	root.Append(NewMutex(tableHandle))
	root.Append(NewEvent(tableHandle))
	root.Append(NewEvent(tableHandle))
	root.Append(NewEvent(tableHandle))
	root.Append(NewField(tableHandle))
	root.Append(NewIndexField(tableHandle))
	root.Append(NewBankField(tableHandle))
	root.Append(&Invocation{
		Generic: Generic{
			op: OpMethodInvocation,
			args: []interface{}{
				NewConst(OpOne, tableHandle, uint64(1)),
				NewConst(OpDwordPrefix, tableHandle, uint64(2)),
			},
		},
	})

	specs := []struct {
		searchType      Type
		keepRecursingFn func(Entity) bool
		wantHits        int
	}{
		{TypeAny, keepRecursing, 27},
		{TypeAny, stopRecursing, 1},
		{
			TypeAny,
			func(ent Entity) bool {
				// Stop recursing after visiting the Invocation entity
				_, isInv := ent.(*Invocation)
				return !isInv
			},
			25,
		},
		{
			TypeAny,
			func(ent Entity) bool {
				// Stop recursing after visiting the first Const entity
				_, isConst := ent.(*Const)
				return !isConst
			},
			26,
		},
		{TypeDevice, keepRecursing, 1},
		{TypeProcessor, keepRecursing, 2},
		{TypePowerResource, keepRecursing, 3},
		{TypeThermalZone, keepRecursing, 4},
		{TypeMethod, keepRecursing, 5},
		{TypeMutex, keepRecursing, 2},
		{TypeEvent, keepRecursing, 3},
		{TypeField, keepRecursing, 1},
		{TypeIndexField, keepRecursing, 1},
		{TypeBankField, keepRecursing, 1},
	}

	for specIndex, spec := range specs {
		var hits int
		Visit(0, root, spec.searchType, func(_ int, obj Entity) bool {
			hits++
			return spec.keepRecursingFn(obj)
		})

		if hits != spec.wantHits {
			t.Errorf("[spec %d] expected visitor to be called %d times; got %d", specIndex, spec.wantHits, hits)
		}
	}
}

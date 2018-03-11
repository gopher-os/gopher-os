package aml

import (
	"fmt"
	"testing"
)

func TestTreeObjectAt(t *testing.T) {
	tree := NewObjectTree()

	root := tree.newObject(pOpIntScopeBlock, 0)
	obj1 := tree.newObject(pOpIntScopeBlock, 1)

	tree.append(root, obj1)

	if got := tree.ObjectAt(obj1.index); got != obj1 {
		t.Fatalf("expected to get back obj1; got %#+v", got)
	}

	if got := tree.ObjectAt(obj1.index + 1); got != nil {
		t.Fatalf("expected to get nil; got %#+v", got)
	}

	tree.free(obj1)
	if got := tree.ObjectAt(obj1.index); got != nil {
		t.Fatalf("expected to get back nil after freeing obj1; got %#+v", got)
	}
}

func TestTreeFreelist(t *testing.T) {
	tree := NewObjectTree()

	root := tree.newObject(pOpIntScopeBlock, 0)
	obj1 := tree.newObject(pOpIntScopeBlock, 1)
	obj2 := tree.newObject(pOpIntScopeBlock, 2)
	obj3 := tree.newObject(pOpIntScopeBlock, 3)

	tree.append(root, obj1)
	tree.append(root, obj2)
	tree.append(root, obj3)

	// By freeing these objects they will be re-used by the following NewObject
	// calls in LIFO order.
	obj2Index := obj2.index
	obj3Index := obj3.index
	tree.free(obj3)
	tree.free(obj2)

	newObj := tree.newObject(pOpIntScopeBlock, 4)
	if newObj.index != obj2Index {
		t.Errorf("expected object index to be %d; got %d", obj2Index, newObj.index)
	}

	newObj = tree.newObject(pOpIntScopeBlock, 4)
	if newObj.index != obj3Index {
		t.Errorf("expected object index to be %d; got %d", obj3Index, newObj.index)
	}

	if tree.freeListHeadIndex != InvalidIndex {
		t.Errorf("expected free list head index to be InvalidIndex; got %d", tree.freeListHeadIndex)
	}
}

func TestTreeFreelistPanic(t *testing.T) {
	tree := NewObjectTree()
	tree.CreateDefaultScopes(42)

	defer func() {
		expErr := "aml.ObjectTree: attempted to free object that still contains argument references"
		if err := recover(); err != expErr {
			t.Fatalf("expected call to Free to panic with: %s; got: %v", expErr, err)
		}
	}()

	// Call should panic as root contains argument references
	tree.free(tree.ObjectAt(0))
}

func TestTreeAppend(t *testing.T) {
	tree := NewObjectTree()

	root := tree.newObject(pOpIntScopeBlock, 0)
	obj1 := tree.newObject(pOpIntScopeBlock, 1)
	obj2 := tree.newObject(pOpIntScopeBlock, 2)
	obj3 := tree.newObject(pOpIntScopeBlock, 3)

	if root.firstArgIndex != InvalidIndex || root.lastArgIndex != InvalidIndex {
		t.Fatal("expected root First/Last arg indices to be InvalidIndex")
	}

	tree.append(root, obj1)
	if root.firstArgIndex != obj1.index || root.lastArgIndex != obj1.index {
		t.Fatal("expected root First/Last arg indices to point to obj1")
	}

	if obj1.parentIndex != root.index {
		t.Fatal("expected obj1 parent index to point to root")
	}

	if obj1.firstArgIndex != InvalidIndex || obj1.lastArgIndex != InvalidIndex {
		t.Fatal("expected obj1 First/Last arg indices to be InvalidIndex")
	}

	// Attach the remaining pointers and follow the links both ways
	tree.append(root, obj2)
	tree.append(root, obj3)

	visitedObjects := 0
	for i := root.firstArgIndex; i != InvalidIndex; i = tree.ObjectAt(i).nextSiblingIndex {
		visitedObjects++
	}

	if want := 3; visitedObjects != want {
		t.Fatalf("expected to visit %d objects in left -> right traversal; visited %d", want, visitedObjects)
	}

	visitedObjects = 0
	for i := root.lastArgIndex; i != InvalidIndex; i = tree.ObjectAt(i).prevSiblingIndex {
		visitedObjects++
	}

	if want := 3; visitedObjects != want {
		t.Fatalf("expected to visit %d objects in right -> left traversal; visited %d", want, visitedObjects)
	}
}

func TestTreeAppendAfter(t *testing.T) {
	tree := NewObjectTree()

	root := tree.newObject(pOpIntScopeBlock, 0)
	obj1 := tree.newObject(pOpIntScopeBlock, 1)
	obj2 := tree.newObject(pOpIntScopeBlock, 2)
	obj3 := tree.newObject(pOpIntScopeBlock, 3)

	if root.firstArgIndex != InvalidIndex || root.lastArgIndex != InvalidIndex {
		t.Fatal("expected root First/Last arg indices to be InvalidIndex")
	}

	tree.append(root, obj1)

	tree.appendAfter(root, obj2, obj1)
	tree.appendAfter(root, obj3, obj1)

	expIndexList := []uint32{obj1.index, obj3.index, obj2.index}

	visitedObjects := 0
	for i := root.firstArgIndex; i != InvalidIndex; i = tree.ObjectAt(i).nextSiblingIndex {
		if i != expIndexList[visitedObjects] {
			t.Fatalf("expected arg %d to have index %d; got %d", visitedObjects, expIndexList[visitedObjects], i)
		}
		visitedObjects++
	}

	if want := 3; visitedObjects != want {
		t.Fatalf("expected to visit %d objects in left -> right traversal; visited %d", want, visitedObjects)
	}
}

func TestTreeDetach(t *testing.T) {
	tree := NewObjectTree()

	root := tree.newObject(pOpIntScopeBlock, 0)
	obj1 := tree.newObject(pOpIntScopeBlock, 1)
	obj2 := tree.newObject(pOpIntScopeBlock, 2)
	obj3 := tree.newObject(pOpIntScopeBlock, 3)

	t.Run("detach in reverse order", func(t *testing.T) {
		tree.append(root, obj1)
		tree.append(root, obj2)
		tree.append(root, obj3)

		tree.detach(root, obj3)
		if root.firstArgIndex != obj1.index || root.lastArgIndex != obj2.index {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", obj1.index, obj2.index, root.firstArgIndex, root.lastArgIndex)
		}

		tree.detach(root, obj2)
		if root.firstArgIndex != obj1.index || root.lastArgIndex != obj1.index {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", obj1.index, obj1.index, root.firstArgIndex, root.lastArgIndex)
		}

		tree.detach(root, obj1)
		if root.firstArgIndex != InvalidIndex || root.lastArgIndex != InvalidIndex {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", InvalidIndex, InvalidIndex, root.firstArgIndex, root.lastArgIndex)
		}
	})

	t.Run("detach in insertion order", func(t *testing.T) {
		tree.append(root, obj1)
		tree.append(root, obj2)
		tree.append(root, obj3)

		tree.detach(root, obj1)
		if root.firstArgIndex != obj2.index || root.lastArgIndex != obj3.index {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", obj2.index, obj3.index, root.firstArgIndex, root.lastArgIndex)
		}

		tree.detach(root, obj2)
		if root.firstArgIndex != obj3.index || root.lastArgIndex != obj3.index {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", obj3.index, obj3.index, root.firstArgIndex, root.lastArgIndex)
		}

		tree.detach(root, obj3)
		if root.firstArgIndex != InvalidIndex || root.lastArgIndex != InvalidIndex {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", InvalidIndex, InvalidIndex, root.firstArgIndex, root.lastArgIndex)
		}
	})

	t.Run("detach middle node and then edges", func(t *testing.T) {
		tree.append(root, obj1)
		tree.append(root, obj2)
		tree.append(root, obj3)

		tree.detach(root, obj2)
		if root.firstArgIndex != obj1.index || root.lastArgIndex != obj3.index {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", obj1.index, obj3.index, root.firstArgIndex, root.lastArgIndex)
		}

		if obj1.nextSiblingIndex != obj3.index {
			t.Fatalf("expected obj1 NextSiblingIndex to be %d; got %d", obj3.index, obj1.nextSiblingIndex)
		}

		if obj3.prevSiblingIndex != obj1.index {
			t.Fatalf("expected obj3 PrevSiblingIndex to be %d; got %d", obj1.index, obj3.prevSiblingIndex)
		}

		tree.detach(root, obj1)
		if root.firstArgIndex != obj3.index || root.lastArgIndex != obj3.index {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", obj3.index, obj3.index, root.firstArgIndex, root.lastArgIndex)
		}

		tree.detach(root, obj3)
		if root.firstArgIndex != InvalidIndex || root.lastArgIndex != InvalidIndex {
			t.Fatalf("unexpected first/last indices: want (%d, %d); got (%d, %d)", InvalidIndex, InvalidIndex, root.firstArgIndex, root.lastArgIndex)
		}
	})
}

func TestFind(t *testing.T) {
	tree, scopeMap := genTestScopes()

	specs := []struct {
		curScope uint32
		expr     string
		want     uint32
	}{
		// Search rules do not apply for these cases
		{
			scopeMap["PCI0"],
			`\`,
			scopeMap[`\`],
		},
		{
			scopeMap["PCI0"],
			"IDE0_ADR",
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"],
			"^^PCI0IDE0_ADR",
			scopeMap["_ADR"],
		},
		{
			scopeMap["IDE0"],
			`\_SB_PCI0IDE0_ADR`,
			scopeMap["_ADR"],
		},
		// Raw multi-segment path (prefix 0x2f, segCount: 3)
		{
			scopeMap["_ADR"],
			fmt.Sprintf("\\%c%c_SB_PCI0IDE0", 0x2f, 0x03),
			scopeMap["IDE0"],
		},
		{
			scopeMap["IDE0"],
			`\_SB_PCI0`,
			scopeMap["PCI0"],
		},
		{
			scopeMap["IDE0"],
			`^`,
			scopeMap["PCI0"],
		},
		// Bad queries
		{
			scopeMap["_SB_"],
			"PCI0USB0_CRS",
			InvalidIndex,
		},
		{
			scopeMap["IDE0"],
			"^^^^^^^^^^^^^^^^^^^",
			InvalidIndex,
		},
		{
			scopeMap["IDE0"],
			`^^^^^^^^^^^FOO`,
			InvalidIndex,
		},
		{
			scopeMap["IDE0"],
			"FOO",
			InvalidIndex,
		},
		{
			scopeMap["IDE0"],
			"",
			InvalidIndex,
		},
		// Incomplete multi-segment path (prefix 0x2f, segCount: 3)
		{
			scopeMap["_ADR"],
			fmt.Sprintf("\\%c%c?", 0x2f, 0x03),
			InvalidIndex,
		},
		// Search rules apply for these cases
		{
			scopeMap["IDE0"],
			"_CRS",
			scopeMap["_CRS"],
		},
	}

	for specIndex, spec := range specs {
		if got := tree.Find(spec.curScope, []byte(spec.expr)); got != spec.want {
			t.Errorf("[spec %d] expected lookup to return index %d; got %d", specIndex, spec.want, got)
		}
	}
}

func TestNumArgs(t *testing.T) {
	tree := NewObjectTree()
	tree.CreateDefaultScopes(42)

	if exp, got := uint32(5), tree.NumArgs(tree.ObjectAt(0)); got != exp {
		t.Fatalf("expected NumArgs(root) to return %d; got %d", exp, got)
	}

	if got := tree.NumArgs(nil); got != 0 {
		t.Fatalf("expected NumArgs(nil) to return 0; got %d", got)
	}
}

func TestArgAt(t *testing.T) {
	tree := NewObjectTree()
	tree.CreateDefaultScopes(42)

	root := tree.ObjectAt(0)
	arg0 := tree.ArgAt(root, 0)
	expName := [amlNameLen]byte{'_', 'G', 'P', 'E'}
	if arg0.name != expName {
		t.Errorf("expected ArgAt(root, 0) to return object with name: %s; got: %s", string(expName[:]), string(arg0.name[:]))
	}

	if got := tree.ArgAt(root, InvalidIndex); got != nil {
		t.Errorf("expected ArgAt(root, InvalidIndex) to return nil; got: %v", got)
	}

	if got := tree.ArgAt(nil, 1); got != nil {
		t.Fatalf("expected ArgAt(nil, x) to return nil; got %#+v", got)
	}
}

func TestClosestNamedAncestor(t *testing.T) {
	tree := NewObjectTree()
	root := tree.newObject(pOpIntScopeBlock, 0)
	obj1 := tree.newObject(pOpMethod, 1)
	obj2 := tree.newObject(pOpIf, 2)
	scope := tree.newObject(pOpScope, 3)
	obj3 := tree.newObject(pOpIf, 4)

	tree.append(root, obj1)
	tree.append(obj1, obj2)
	tree.append(obj2, scope)
	tree.append(scope, obj3)

	if got := tree.ClosestNamedAncestor(obj3); got != InvalidIndex {
		t.Errorf("expected ClosestNamedAncestor to return InvalidIndex; got %d", got)
	}

	// No parent exists
	if got := tree.ClosestNamedAncestor(root); got != InvalidIndex {
		t.Errorf("expected ClosestNamedAncestor to return InvalidIndex; got %d", got)
	}

	// Make the root a non-named object and call ClosestNamedAncestor on its child
	root.opcode = pOpAdd
	root.infoIndex = pOpcodeTableIndex(root.opcode, false)
	if got := tree.ClosestNamedAncestor(obj1); got != InvalidIndex {
		t.Errorf("expected ClosestNamedAncestor to return InvalidIndex; got %d", got)
	}

	if got := tree.ClosestNamedAncestor(nil); got != InvalidIndex {
		t.Fatalf("expected ClosestNamedAncestor(nil) to return InvalidIndex; got %d", got)
	}
}

func genTestScopes() (*ObjectTree, map[string]uint32) {
	// Setup the example tree from page 252 of the acpi 6.2 spec
	// \
	//  SB
	//    \
	//     PCI0
	//         | _CRS
	//         \
	//          IDE0
	//              | _ADR

	tree := NewObjectTree()

	root := tree.newNamedObject(pOpIntScopeBlock, 0, [4]byte{'\\'})
	pci := tree.newNamedObject(pOpIntScopeBlock, 0, [4]byte{'P', 'C', 'I', '0'})
	ide := tree.newNamedObject(pOpIntScopeBlock, 0, [4]byte{'I', 'D', 'E', '0'})
	sb := tree.newNamedObject(pOpIntScopeBlock, 0, [4]byte{'_', 'S', 'B', '_'})

	crs := tree.newNamedObject(pOpIntScopeBlock, 0, [4]byte{'_', 'C', 'R', 'S'})
	adr := tree.newNamedObject(pOpIntScopeBlock, 0, [4]byte{'_', 'A', 'D', 'R'})

	// Setup tree
	tree.append(root, sb)
	tree.append(sb, pci)
	tree.append(pci, crs)
	tree.append(pci, ide)
	tree.append(ide, adr)

	return tree, map[string]uint32{
		"IDE0": ide.index,
		"PCI0": pci.index,
		"_SB_": sb.index,
		"\\":   root.index,
		"_ADR": adr.index,
		"_CRS": crs.index,
	}
}

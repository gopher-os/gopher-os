package aml

import (
	"bytes"
	"gopheros/kernel/kfmt"
	"io"
)

const (
	// InvalidIndex is a sentinel value used by ObjectTree to indicate that
	// a returned object index is not valid.
	InvalidIndex uint32 = (1 << 32) - 1

	// The size of AML name identifiers in bytes.
	amlNameLen = 4
)

// fieldElement groups together information about a field element. This
// information can also be obtained by scanning a field element's siblings but
// it is summarized in this structure for convenience.
type fieldElement struct {
	// The offset in the address space defined by parent field.
	offset uint32

	// The width of this field element in bits.
	width uint32

	accessLength uint8
	_pad         [3]uint8

	accessType   uint8
	accessAttrib uint8
	lockType     uint8
	updateType   uint8

	// The index of an Object to use as a connection target. Used only for
	// GenericSerialBus and GeneralPurposeIO operation regions.
	connectionIndex uint32

	// The index of the Field (pOpField, pOpIndexField or pOpBankField) that this
	// field element belongs to. According to the spec, named fields appear
	// at the same scope as their parent.
	fieldIndex uint32
}

// Object describes an entity encoded in the AML bytestream.
type Object struct {
	// The AML ocode that describes this oBject.
	opcode uint16

	// The Index in the opcodeTable for this opcode
	infoIndex uint8

	// The table handle which contains this entity.
	tableHandle uint8

	// Named AML entities provide a fixed-width name which is padded by '_' chars.
	name [amlNameLen]byte

	// The following indices refer to other Objects in the ObjectTree
	// that allocated this Object instance. Uninitialized indices are
	// represented via a math.MaxUint32 value.
	index            uint32
	parentIndex      uint32
	prevSiblingIndex uint32
	nextSiblingIndex uint32
	firstArgIndex    uint32
	lastArgIndex     uint32

	// The byte offset in the AML stream where this opcode is defined.
	amlOffset uint32

	// A non-zero value for pkgEnd indicates that this opcode requires deferred
	// parsing due to its potentially ambiguous contents.
	pkgEnd uint32

	// A value placeholder for entites that contain values (e.g. int
	// or string constants byte slices e.t.c)
	value interface{}
}

// ObjectTree is a structure that contains a tree of AML entities where each
// entity is allocated from a contiguous Object pool. Index #0 of the pool
// contains the root scope ('\') of the AML tree.
type ObjectTree struct {
	objPool           []*Object
	freeListHeadIndex uint32
}

// NewObjectTree returns a new ObjectTree instance.
func NewObjectTree() *ObjectTree {
	return &ObjectTree{
		freeListHeadIndex: InvalidIndex,
	}
}

// CreateDefaultScopes populates the Object pool with the default scopes
// specified by the ACPI standard:
//
//   +-[\] (Root scope)
//      +- [_GPE] (General events in GPE register block)
//      +- [_PR_] (ACPI 1.0 processor namespace)
//      +- [_SB_] (System bus with all device objects)
//      +- [_SI_] (System indicators)
//      +- [_TZ_] (ACPI 1.0 thermal zone namespace)
func (tree *ObjectTree) CreateDefaultScopes(tableHandle uint8) {
	root := tree.newNamedObject(pOpIntScopeBlock, tableHandle, [amlNameLen]byte{'\\'})
	tree.append(root, tree.newNamedObject(pOpIntScopeBlock, tableHandle, [amlNameLen]byte{'_', 'G', 'P', 'E'})) // General events in GPE register block
	tree.append(root, tree.newNamedObject(pOpIntScopeBlock, tableHandle, [amlNameLen]byte{'_', 'P', 'R', '_'})) // ACPI 1.0 processor namespace
	tree.append(root, tree.newNamedObject(pOpIntScopeBlock, tableHandle, [amlNameLen]byte{'_', 'S', 'B', '_'})) // System bus with all device objects
	tree.append(root, tree.newNamedObject(pOpIntScopeBlock, tableHandle, [amlNameLen]byte{'_', 'S', 'I', '_'})) // System indicators
	tree.append(root, tree.newNamedObject(pOpIntScopeBlock, tableHandle, [amlNameLen]byte{'_', 'T', 'Z', '_'})) // ACPI 1.0 thermal zone namespace
}

// newObject allocates a new Object from the Object pool, populates its
// contents and returns back a pointer to it.
func (tree *ObjectTree) newObject(opcode uint16, tableHandle uint8) *Object {
	var obj *Object

	// Check the free list first
	if tree.freeListHeadIndex != InvalidIndex {
		obj = tree.objPool[tree.freeListHeadIndex]
		tree.freeListHeadIndex = obj.nextSiblingIndex
	} else {
		// Allocate new object and attach it to the pool
		obj = new(Object)
		obj.index = uint32(len(tree.objPool))
		tree.objPool = append(tree.objPool, obj)
	}

	obj.opcode = opcode
	obj.infoIndex = pOpcodeTableIndex(opcode, true)
	obj.tableHandle = tableHandle
	obj.parentIndex = InvalidIndex
	obj.prevSiblingIndex = InvalidIndex
	obj.nextSiblingIndex = InvalidIndex
	obj.firstArgIndex = InvalidIndex
	obj.lastArgIndex = InvalidIndex
	obj.value = nil

	return obj
}

// newNamedObject allocates a new Object from the Object pool, populates its
// name and returns back a pointer to it.
func (tree *ObjectTree) newNamedObject(opcode uint16, tableHandle uint8, name [amlNameLen]byte) *Object {
	obj := tree.newObject(opcode, tableHandle)
	obj.name = name
	return obj
}

// append appends arg to obj's argument list.
func (tree *ObjectTree) append(obj, arg *Object) {
	arg.parentIndex = obj.index

	if obj.lastArgIndex == InvalidIndex {
		obj.firstArgIndex = arg.index
		obj.lastArgIndex = arg.index
		return
	}

	LastArg := tree.ObjectAt(obj.lastArgIndex)
	LastArg.nextSiblingIndex = arg.index
	arg.prevSiblingIndex = LastArg.index
	arg.nextSiblingIndex = InvalidIndex
	obj.lastArgIndex = arg.index
}

// appendAfter appends arg to obj's argument list after nextTo.
func (tree *ObjectTree) appendAfter(obj, arg, nextTo *Object) {
	// nextTo is the last arg of obj; this is equivalent to a regular append
	if nextTo.nextSiblingIndex == InvalidIndex {
		tree.append(obj, arg)
		return
	}

	arg.parentIndex = obj.index
	arg.prevSiblingIndex = nextTo.index
	arg.nextSiblingIndex = nextTo.nextSiblingIndex

	tree.ObjectAt(arg.nextSiblingIndex).prevSiblingIndex = arg.index
	nextTo.nextSiblingIndex = arg.index
}

// free appends obj to the tree's free object list allowing it to be re-used by
// future calls to NewObject and NewNamedObject. Callers must ensure that any
// held pointers to the object are not used after calling free.
func (tree *ObjectTree) free(obj *Object) {
	if obj.parentIndex != InvalidIndex {
		tree.detach(tree.ObjectAt(obj.parentIndex), obj)
	}

	if obj.firstArgIndex != InvalidIndex || obj.lastArgIndex != InvalidIndex {
		panic("aml.ObjectTree: attempted to free object that still contains argument references")
	}

	// Push the object to the top of the free list and change its opcode to
	// indicate this is a freed node
	obj.opcode = pOpIntFreedObject
	obj.nextSiblingIndex = tree.freeListHeadIndex
	tree.freeListHeadIndex = obj.index
}

// detach detaches arg from obj's argument list.
func (tree *ObjectTree) detach(obj, arg *Object) {
	if obj.firstArgIndex == arg.index {
		obj.firstArgIndex = arg.nextSiblingIndex
	}

	if obj.lastArgIndex == arg.index {
		obj.lastArgIndex = arg.prevSiblingIndex
	}

	if arg.nextSiblingIndex != InvalidIndex {
		tree.ObjectAt(arg.nextSiblingIndex).prevSiblingIndex = arg.prevSiblingIndex
	}

	if arg.prevSiblingIndex != InvalidIndex {
		tree.ObjectAt(arg.prevSiblingIndex).nextSiblingIndex = arg.nextSiblingIndex
	}

	arg.prevSiblingIndex = InvalidIndex
	arg.nextSiblingIndex = InvalidIndex
	arg.parentIndex = InvalidIndex
}

// ObjectAt returns a pointer to the Object at the specified index or nil if
// no object with this index exists inside the object tree.
func (tree *ObjectTree) ObjectAt(index uint32) *Object {
	if index >= uint32(len(tree.objPool)) {
		return nil
	}
	obj := tree.objPool[index]
	if obj.opcode == pOpIntFreedObject {
		return nil
	}

	return obj
}

// Find attempts to resolve the given expression into an Object using the rules
// specified in page 252 of the ACPI 6.2 spec:
//
// There are two types of namespace paths: an absolute namespace path (that is,
// one that starts with a ‘\’ prefix), and a relative namespace path (that is,
// one that is relative to the current namespace). The namespace search rules
// discussed above, only apply to single NameSeg paths, which is a relative
// namespace path. For those relative name paths that contain multiple NameSegs
// or Parent Prefixes, ‘^’, the search rules do not apply. If the search rules
// do not apply to a relative namespace path, the namespace object is looked up
// relative to the current namespace
func (tree *ObjectTree) Find(scopeIndex uint32, expr []byte) uint32 {
	exprLen := len(expr)
	if exprLen == 0 || scopeIndex == InvalidIndex {
		return InvalidIndex
	}

	switch {
	case expr[0] == '\\': // relative to the root scope
		// Name was just `\`; this matches the root namespace
		if exprLen == 1 {
			return 0
		}

		return tree.findRelative(0, expr[1:])
	case expr[0] == '^': // relative to the parent scope(s)
		for startIndex := 0; startIndex < exprLen; startIndex++ {
			switch expr[startIndex] {
			case '^':
				// Mpve up one parent. If we were at the root scope
				// then the lookup failed.
				if scopeIndex = tree.ObjectAt(scopeIndex).parentIndex; scopeIndex == InvalidIndex {
					return InvalidIndex
				}
			default:
				// Found the start of the name. Look it up relative to current scope
				return tree.findRelative(scopeIndex, expr[startIndex:])
			}
		}

		// Name was just a sequence of '^'; this matches the last scopeIndex value
		return scopeIndex
	case exprLen > amlNameLen:
		// The expression consists of multiple name segments joined together (e.g. FOOFBAR0)
		// In this case we need to apply relative lookup rules for FOOF.BAR0
		return tree.findRelative(scopeIndex, expr)
	default:
		// expr is a simple name. According to the spec, we need to
		// search for it in this scope and all its parent scopes till
		// we reach the root.
		for nextScopeIndex := scopeIndex; nextScopeIndex != InvalidIndex; nextScopeIndex = tree.ObjectAt(nextScopeIndex).parentIndex {
			scopeObj := tree.ObjectAt(nextScopeIndex)
		checkNextSibling:
			for nextIndex := scopeObj.firstArgIndex; nextIndex != InvalidIndex; nextIndex = tree.ObjectAt(nextIndex).nextSiblingIndex {
				obj := tree.ObjectAt(nextIndex)
				for byteIndex := 0; byteIndex < amlNameLen; byteIndex++ {
					if expr[byteIndex] != obj.name[byteIndex] {
						continue checkNextSibling
					}
				}

				// Found match
				return obj.index
			}
		}
	}

	// Not found
	return InvalidIndex
}

// findRelative attempts to resolve an object using relative scope lookup rules.
func (tree *ObjectTree) findRelative(scopeIndex uint32, expr []byte) uint32 {
	exprLen := len(expr)

nextSegment:
	for segIndex := 0; segIndex < exprLen; segIndex += amlNameLen {
		// If expr contains a dual or multinamed path then we may encounter special
		// prefix chars in the stream (the parser extracts the raw data). In this
		// case skip over them.
		for ; segIndex < exprLen && expr[segIndex] != '_' && (expr[segIndex] < 'A' || expr[segIndex] > 'Z'); segIndex++ {
		}
		if segIndex >= exprLen {
			return InvalidIndex
		}

		// Search current scope for an entity matching the next name segment
		scopeObj := tree.ObjectAt(scopeIndex)

	checkNextSibling:
		for nextIndex := scopeObj.firstArgIndex; nextIndex != InvalidIndex; nextIndex = tree.ObjectAt(nextIndex).nextSiblingIndex {
			obj := tree.ObjectAt(nextIndex)
			for byteIndex := 0; byteIndex < amlNameLen; byteIndex++ {
				if expr[segIndex+byteIndex] != obj.name[byteIndex] {
					continue checkNextSibling
				}
			}

			// Found match; set match as the next scope index and
			// try to match the next segment
			scopeIndex = nextIndex
			continue nextSegment
		}

		// Failed to match next segment. Lookup failed
		return InvalidIndex
	}

	// scopeIndex contains the index of the last matched name in the expression
	// which is the target we were looking for.
	return scopeIndex
}

// ClosestNamedAncestor returns the index of the first named object that is an
// ancestor of obj. If any of obj's parents are unresolved scope directives
// then the call will return InvalidIndex.
func (tree *ObjectTree) ClosestNamedAncestor(obj *Object) uint32 {
	if obj == nil {
		return InvalidIndex
	}

	for ancestorIndex := obj.parentIndex; ancestorIndex != InvalidIndex; {
		ancestor := tree.ObjectAt(ancestorIndex)
		if ancestor.opcode == pOpScope {
			break
		}

		if pOpcodeTable[ancestor.infoIndex].flags&pOpFlagNamed != 0 {
			return ancestorIndex
		}

		ancestorIndex = ancestor.parentIndex
	}

	return InvalidIndex
}

// NumArgs returns the number of arguments contained in obj.
func (tree *ObjectTree) NumArgs(obj *Object) uint32 {
	if obj == nil {
		return 0
	}

	var argCount uint32
	for siblingIndex := obj.firstArgIndex; siblingIndex != InvalidIndex; siblingIndex = tree.ObjectAt(siblingIndex).nextSiblingIndex {
		argCount++
	}

	return argCount
}

// ArgAt returns a pointer to obj's arg located at index.
func (tree *ObjectTree) ArgAt(obj *Object, index uint32) *Object {
	if obj == nil {
		return nil
	}
	for argIndex, siblingIndex := uint32(0), obj.firstArgIndex; siblingIndex != InvalidIndex; argIndex, siblingIndex = argIndex+1, tree.ObjectAt(siblingIndex).nextSiblingIndex {
		if argIndex == index {
			return tree.ObjectAt(siblingIndex)
		}
	}

	return nil
}

// PrettyPrint outputs a pretty-printed version of the AML tree to w.
func (tree *ObjectTree) PrettyPrint(w io.Writer) {
	if len(tree.objPool) != 0 {
		var padBuf bytes.Buffer
		padBuf.WriteByte(' ')
		tree.toString(w, &padBuf, 0)
	}
}

func (tree *ObjectTree) toString(w io.Writer, padBuf *bytes.Buffer, index uint32) {
	curObj := tree.ObjectAt(index)

	_, _ = w.Write(padBuf.Bytes())
	kfmt.Fprintf(w, "+- [%s", pOpcodeName(curObj.opcode))

	if name := nameOf(curObj); len(name) != 0 {
		kfmt.Fprintf(w, ", name: \"%s\"", name)
	}

	if curObj.opcode == pOpMethod {
		kfmt.Fprintf(w, ", argCount: %d", uint8(tree.ArgAt(curObj, 1).value.(uint64)&0x7))
	}

	kfmt.Fprintf(w, ", table: %d, index: %d, offset: 0x%x", curObj.tableHandle, curObj.index, curObj.amlOffset)
	kfmt.Fprintf(w, "]")

	if curObj.opcode == pOpIntMethodCall {
		methodObj := tree.ObjectAt(curObj.value.(uint32))
		argCount := uint8(tree.ArgAt(methodObj, 1).value.(uint64) & 0x7)
		kfmt.Fprintf(w, " -> [call to \"%s\", argCount: %d, table: %d, index: %d, offset: 0x%x]", methodObj.name[:], argCount, methodObj.tableHandle, methodObj.index, methodObj.amlOffset)
	} else if curObj.opcode == pOpIntResolvedNamePath {
		resolvedObj := tree.ObjectAt(curObj.value.(uint32))
		kfmt.Fprintf(w, " -> [resolved to \"%s\", table: %d, index: %d, offset: 0x%x]", nameOf(resolvedObj), resolvedObj.tableHandle, resolvedObj.index, resolvedObj.amlOffset)
	} else if curObj.opcode == pOpIntNamedField {
		field := curObj.value.(*fieldElement)
		kfmt.Fprintf(w, " -> [field index: %d, offset(bytes): 0x%x, width(bits): 0x%x, accType: ", field.fieldIndex, field.offset, field.width)
		switch field.accessType {
		case 0x00:
			kfmt.Fprintf(w, "Any")
		case 0x01:
			kfmt.Fprintf(w, "Byte")
		case 0x02:
			kfmt.Fprintf(w, "Word")
		case 0x03:
			kfmt.Fprintf(w, "Dword")
		case 0x04:
			kfmt.Fprintf(w, "Qword")
		case 0x05:
			kfmt.Fprintf(w, "Buffer, accAttr: ")
			switch field.accessAttrib {
			case 0x02:
				kfmt.Fprintf(w, "Quick")
			case 0x04:
				kfmt.Fprintf(w, "SendReceive")
			case 0x06:
				kfmt.Fprintf(w, "Byte")
			case 0x08:
				kfmt.Fprintf(w, "Word")
			case 0x0a:
				kfmt.Fprintf(w, "Block")
			case 0x0b:
				kfmt.Fprintf(w, "Bytes(0x%x)", field.accessLength)
			case 0x0c:
				kfmt.Fprintf(w, "ProcessCall")
			case 0x0d:
				kfmt.Fprintf(w, "BlockProcessCall")
			case 0x0e:
				kfmt.Fprintf(w, "RawBytes(0x%x)", field.accessLength)
			case 0x0f:
				kfmt.Fprintf(w, "RawProcessBytes(0x%x)", field.accessLength)
			}
			/*
				case 0x40:
					kfmt.Fprintf(w, "Bytes(0x%x)", field.AccessAttrib)
				case 0x80:
					kfmt.Fprintf(w, "RawBytes(0x%x)", field.AccessAttrib)
				case 0xc0:
					kfmt.Fprintf(w, "RawProcessBytes(0x%x)", field.AccessAttrib)
			*/
		}

		kfmt.Fprintf(w, ", lockType: ")
		switch field.lockType {
		case 0x00:
			kfmt.Fprintf(w, "NoLock")
		case 0x01:
			kfmt.Fprintf(w, "Lock")
		}

		kfmt.Fprintf(w, ", updateType: ")
		switch field.updateType {
		case 0x00:
			kfmt.Fprintf(w, "Preserve")
		case 0x01:
			kfmt.Fprintf(w, "WriteAsOnes")
		case 0x02:
			kfmt.Fprintf(w, "WriteAsZeroes")
		}

		switch field.connectionIndex {
		case InvalidIndex:
			kfmt.Fprintf(w, ", connection: -]")
		default:
			kfmt.Fprintf(w, ", connection: index %d]", field.connectionIndex)
		}
	} else if curObj.opcode == pOpStringPrefix {
		kfmt.Fprintf(w, " -> [string value: \"%s\"]", curObj.value.([]byte))
	} else if curObj.opcode == pOpIntNamePath {
		kfmt.Fprintf(w, " -> [namepath: \"%s\"]", curObj.value.([]byte))
	} else if curObj.value != nil {
		switch v := curObj.value.(type) {
		case uint64:
			kfmt.Fprintf(w, " -> [num value; dec: %d, hex: 0x%x]", v, v)

			// If this is an encoded EISA id convert it back to a string
			if curObj.opcode == pOpDwordPrefix && tree.ObjectAt(curObj.parentIndex).name == [amlNameLen]byte{'_', 'H', 'I', 'D'} {
				// Poor-man's ntohl
				id := uint32((v>>24)&0xff) |
					uint32((v>>16)&0xff)<<8 |
					uint32((v>>8)&0xff)<<16 |
					uint32(v&0xff)<<24

				var eisaID = [7]byte{
					'@' + (byte)((id>>26)&0x1f),
					'@' + (byte)((id>>21)&0x1f),
					'@' + (byte)((id>>16)&0x1f),
					hexToASCII(id >> 12),
					hexToASCII(id >> 8),
					hexToASCII(id >> 4),
					hexToASCII(id),
				}

				kfmt.Fprintf(w, " [EISA: \"%s\"]", eisaID[:])
			}
		case []byte:

			kfmt.Fprintf(w, " -> [bytelist value; len: %d; data: [", len(v))
			for i, b := range v {
				if i != 0 {
					kfmt.Fprintf(w, ", ")
				}
				kfmt.Fprintf(w, "0x%x", uint8(b))
			}
			kfmt.Fprintf(w, "]]")
		}
	}

	kfmt.Fprintf(w, "\n")

	padLen := padBuf.Len()
	if curObj.nextSiblingIndex == InvalidIndex {
		padBuf.WriteByte(' ')
	} else {
		padBuf.WriteByte('|')
	}
	padBuf.WriteByte(' ')
	padBuf.WriteByte(' ')

	for argIndex := curObj.firstArgIndex; argIndex != InvalidIndex; argIndex = tree.ObjectAt(argIndex).nextSiblingIndex {
		tree.toString(w, padBuf, argIndex)
	}

	padBuf.Truncate(padLen)
}

func hexToASCII(val uint32) byte {
	v := byte(val & 0xf)
	if v <= 9 {
		return '0' + v
	}

	return 'A' + (v - 0xa)
}

func nameOf(obj *Object) []byte {
	var nameStartIndex, nameEndIndex int
	for ; nameStartIndex < amlNameLen; nameStartIndex++ {
		if ch := obj.name[nameStartIndex]; (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '\\' {
			break
		}
	}
	for nameEndIndex = nameStartIndex; nameEndIndex < amlNameLen; nameEndIndex++ {
		if ch := obj.name[nameEndIndex]; !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '\\') {
			break
		}
	}

	if nameStartIndex != amlNameLen {
		return obj.name[nameStartIndex:nameEndIndex]
	}

	return nil
}

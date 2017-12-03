package aml

import "strings"

// Visitor is a function invoked by the VM for each AML tree entity that matches
// a particular type. The return value controls whether the children of this
// entity should also be visited.
type Visitor func(depth int, obj Entity) (keepRecursing bool)

// EntityType defines the type of entity that visitors should inspect.
type EntityType uint8

// The list of supported EntityType values. EntityTypeAny works as a wildcard
// allowing the visitor to inspect all entities in the AML tree.
const (
	EntityTypeAny EntityType = iota
	EntityTypeDevice
	EntityTypeProcessor
	EntityTypePowerResource
	EntityTypeThermalZone
	EntityTypeMethod
)

// scopeVisit descends a scope hierarchy and invokes visitorFn for each entity
// that matches entType.
func scopeVisit(depth int, ent Entity, entType EntityType, visitorFn Visitor) bool {
	op := ent.getOpcode()
	switch {
	case (entType == EntityTypeAny) ||
		(entType == EntityTypeDevice && op == opDevice) ||
		(entType == EntityTypeProcessor && op == opProcessor) ||
		(entType == EntityTypePowerResource && op == opPowerRes) ||
		(entType == EntityTypeThermalZone && op == opThermalZone) ||
		(entType == EntityTypeMethod && op == opMethod):
		// If the visitor returned false we should not visit the children
		if !visitorFn(depth, ent) {
			return false
		}

		// Visit any args that are also entities
		for _, arg := range ent.getArgs() {
			if argEnt, isEnt := arg.(Entity); isEnt && !scopeVisit(depth+1, argEnt, entType, visitorFn) {
				return false
			}
		}
	}

	switch typ := ent.(type) {
	case ScopeEntity:
		// If the entity defines a scope we need to visit the child entities.
		for _, child := range typ.Children() {
			_ = scopeVisit(depth+1, child, entType, visitorFn)
		}
	}

	return true
}

// scopeResolvePath examines a path expression and attempts to break it down
// into a parent and child segment. The parent segment is looked up via the
// regular scope rules specified in page 252 of the ACPI 6.2 spec. If the
// parent scope is found then the function returns back the parent entity and
// the name of the child that should be appended to it. If the expression
// lookup fails then the function returns nil, "".
func scopeResolvePath(curScope, rootScope ScopeEntity, expr string) (parent ScopeEntity, name string) {
	if len(expr) <= 1 {
		return nil, ""
	}

	// Pattern looks like \FOO or ^+BAR or BAZ (relative to curScope)
	lastDotIndex := strings.LastIndexByte(expr, '.')
	if lastDotIndex == -1 {
		switch expr[0] {
		case '\\':
			return rootScope, expr[1:]
		case '^':
			lastHatIndex := strings.LastIndexByte(expr, '^')
			if target := scopeFind(curScope, rootScope, expr[:lastHatIndex+1]); target != nil {
				return target.(ScopeEntity), expr[lastHatIndex+1:]
			}

			return nil, ""
		default:
			return curScope, expr
		}
	}

	// Pattern looks like: \FOO.BAR.BAZ or ^+FOO.BAR.BAZ or FOO.BAR.BAZ
	if target := scopeFind(curScope, rootScope, expr[:lastDotIndex]); target != nil {
		return target.(ScopeEntity), expr[lastDotIndex+1:]
	}

	return nil, ""
}

// scopeFind attempts to find an object with the given name using the rules
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
func scopeFind(curScope, rootScope ScopeEntity, name string) Entity {
	nameLen := len(name)
	if nameLen == 0 {
		return nil
	}

	switch {
	case name[0] == '\\': // relative to the root scope
		if nameLen > 1 {
			return scopeFindRelative(rootScope, name[1:])
		}

		// Name was just `\`; this matches the root namespace
		return rootScope
	case name[0] == '^': // relative to the parent scope(s)
		for startIndex := 0; startIndex < nameLen; startIndex++ {
			switch name[startIndex] {
			case '^':
				curScope = curScope.Parent()

				// No parent to visit
				if curScope == nil {
					return nil
				}
			default:
				// Found the start of the name. Look it up relative to curNs
				return scopeFindRelative(curScope, name[startIndex:])
			}
		}

		// Name was just a sequence of '^'; this matches the last curScope value
		return curScope
	case strings.ContainsRune(name, '.'):
		// If the name contains any '.' then we still need to look it
		// up relative to the current scope
		return scopeFindRelative(curScope, name)
	default:
		// We can apply the search rules described by the spec
		for s := curScope; s != nil; s = s.Parent() {
			for _, child := range s.Children() {
				if child.Name() == name {
					return child
				}
			}
		}
	}

	// Not found
	return nil
}

// scopeFindRelative returns the Entity referenced by path relative
// to the provided Namespace. If the name contains dots, each segment
// is used to access a nested namespace. If the path does not point
// to a NamedObject then lookupRelativeTo returns back nil.
func scopeFindRelative(ns ScopeEntity, path string) Entity {
	var matchName string
matchNextPathSegment:
	for {
		dotSepIndex := strings.IndexRune(path, '.')
		if dotSepIndex != -1 {
			matchName = path[:dotSepIndex]
			path = path[dotSepIndex+1:]

			// Search for a scoped child named "matchName"
			for _, child := range ns.Children() {
				childNs, ok := child.(ScopeEntity)
				if !ok {
					continue
				}

				if childNs.Name() == matchName {
					ns = childNs
					continue matchNextPathSegment
				}
			}
		} else {
			// Search for a child named "name"
			for _, child := range ns.Children() {
				if child.Name() == path {
					return child
				}
			}
		}

		// Next segment in the path was not found or last segment not found
		break
	}

	return nil
}

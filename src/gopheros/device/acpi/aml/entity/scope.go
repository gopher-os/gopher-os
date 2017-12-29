package entity

import "strings"

// ResolveScopedPath examines a path expression and attempts to break it down
// into a parent and child segment. The parent segment is looked up via the
// regular scope rules specified in page 252 of the ACPI 6.2 spec. If the
// parent scope is found then the function returns back the parent entity and
// the name of the child that should be appended to it. If the expression
// lookup fails then the function returns nil, "".
func ResolveScopedPath(curScope, rootScope Container, expr string) (parent Container, name string) {
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
			if target := FindInScope(curScope, rootScope, expr[:lastHatIndex+1]); target != nil {
				return target.(Container), expr[lastHatIndex+1:]
			}

			return nil, ""
		default:
			return curScope, expr
		}
	}

	// Pattern looks like: \FOO.BAR.BAZ or ^+FOO.BAR.BAZ or FOO.BAR.BAZ
	if target := FindInScope(curScope, rootScope, expr[:lastDotIndex]); target != nil {
		return target.(Container), expr[lastDotIndex+1:]
	}

	return nil, ""
}

// FindInScope attempts to find an object with the given name using the rules
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
func FindInScope(curScope, rootScope Container, name string) Entity {
	nameLen := len(name)
	if nameLen == 0 {
		return nil
	}

	switch {
	case name[0] == '\\': // relative to the root scope
		if nameLen > 1 {
			return findRelativeToScope(rootScope, name[1:])
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
				return findRelativeToScope(curScope, name[startIndex:])
			}
		}

		// Name was just a sequence of '^'; this matches the last curScope value
		return curScope
	case strings.ContainsRune(name, '.'):
		// If the name contains any '.' then we still need to look it
		// up relative to the current scope
		return findRelativeToScope(curScope, name)
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

// findRelativeToScope returns the Entity referenced by path relative
// to the provided Namespace. If the name contains dots, each segment
// is used to access a nested namespace. If the path does not point
// to a NamedObject then lookupRelativeTo returns back nil.
func findRelativeToScope(ns Container, path string) Entity {
	var matchName string
matchNextPathSegment:
	for {
		dotSepIndex := strings.IndexRune(path, '.')
		if dotSepIndex != -1 {
			matchName = path[:dotSepIndex]
			path = path[dotSepIndex+1:]

			// Search for a scoped child named "matchName"
			for _, child := range ns.Children() {
				if childNs, ok := child.(Container); ok && childNs.Name() == matchName {
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

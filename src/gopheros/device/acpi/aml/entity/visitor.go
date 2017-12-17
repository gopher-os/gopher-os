package entity

// Visitor is a function invoked by the VM for each AML tree entity that matches
// a particular type. The return value controls whether the children of this
// entity should also be visited.
type Visitor func(depth int, obj Entity) (keepRecursing bool)

// Type defines the type of entity that visitors should inspect.
type Type uint8

// The list of supported Type values. TypeAny works as a wildcard
// allowing the visitor to inspect all entities in the AML tree.
const (
	TypeAny Type = iota
	TypeDevice
	TypeProcessor
	TypePowerResource
	TypeThermalZone
	TypeMethod
	TypeMutex
	TypeEvent
	TypeField
	TypeIndexField
	TypeBankField
)

// Visit descends a scope hierarchy and invokes visitorFn for each entity
// that matches entType.
func Visit(depth int, ent Entity, entType Type, visitorFn Visitor) bool {
	op := ent.Opcode()
	switch {
	case (entType == TypeAny) ||
		(entType == TypeDevice && op == OpDevice) ||
		(entType == TypeProcessor && op == OpProcessor) ||
		(entType == TypePowerResource && op == OpPowerRes) ||
		(entType == TypeThermalZone && op == OpThermalZone) ||
		(entType == TypeMethod && op == OpMethod) ||
		(entType == TypeMutex && op == OpMutex) ||
		(entType == TypeEvent && op == OpEvent) ||
		(entType == TypeField && op == OpField) ||
		(entType == TypeIndexField && op == OpIndexField) ||
		(entType == TypeBankField && op == OpBankField):
		// If the visitor returned false we should not visit the children
		if !visitorFn(depth, ent) {
			return false
		}

		// Visit any args that are also entities
		for _, arg := range ent.Args() {
			if argEnt, isEnt := arg.(Entity); isEnt && !Visit(depth+1, argEnt, entType, visitorFn) {
				return false
			}
		}
	}

	// If the entity defines a scope we need to visit the child entities.
	if container, isContainer := ent.(Container); isContainer {
		for _, child := range container.Children() {
			_ = Visit(depth+1, child, entType, visitorFn)
		}
	}

	return true
}

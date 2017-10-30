package aml

// vmLoad returns the value contained inside arg. To obtain the actual stored
// value vmLoad will automatically peek into const entities and lookup
// local/global args.
func vmLoad(ctx *execContext, arg interface{}) (interface{}, *Error) {
	// We need to keep evaluating types till we reach a type that can be
	// returned.  For example, a local arg may contain a const entity for
	// which we want to fetch the contained value.
	for {
		switch typ := arg.(type) {
		case *constEntity:
			arg = typ.val
		case Entity:
			op := typ.getOpcode()
			switch {
			case opIsLocalArg(op):
				arg = ctx.localArg[op-opLocal0]
			case opIsMethodArg(op):
				arg = ctx.methodArg[op-opArg0]
			default:
				return nil, &Error{message: "readArg: unsupported entity type: " + op.String()}
			}
		case bool:
			// Convert boolean results to ints so they can be used
			// by the ALU comparators.
			if typ {
				return uint64(1), nil
			}
			return uint64(0), nil
		default:
			return typ, nil
		}
	}
}

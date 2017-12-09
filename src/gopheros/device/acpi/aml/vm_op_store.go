package aml

// Args: src dst
// Store src into dst applying any required conversion.
func vmOpStore(ctx *execContext, ent Entity) *Error {
	args := ent.getArgs()
	if len(args) != 2 {
		return errArgIndexOutOfBounds
	}

	val, err := vmLoad(ctx, args[0])
	if err != nil {
		return err
	}

	return vmStore(ctx, val, args[1])
}

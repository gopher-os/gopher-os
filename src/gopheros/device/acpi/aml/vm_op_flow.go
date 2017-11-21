package aml

// Args: val
// Set val as the return value in ctx and change the ctrlFlow
// type to ctrlFlowTypeFnReturn.
func vmOpReturn(ctx *execContext, ent Entity) *Error {
	args := ent.getArgs()
	if len(args) != 1 {
		return errArgIndexOutOfBounds
	}

	var err *Error
	ctx.ctrlFlow = ctrlFlowTypeFnReturn
	ctx.retVal, err = vmLoad(ctx, args[0])
	return err
}

package aml

// Args: left, right, store?
// Returns: left + right
func vmOpAdd(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left + right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args: left, right, store?
// Returns: left - right
func vmOpSubtract(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left - right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args: left, store?
// Returns: left + 1
// Stores: left <= left + 1
func vmOpIncrement(ctx *execContext, ent Entity) *Error {
	var (
		left uint64
		err  *Error
	)

	if left, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	// The result is stored back into the left operand
	ctx.retVal = left + 1
	if err := vmStore(ctx, ctx.retVal, ent.getArgs()[0]); err != nil {
		return err
	}
	return vmCondStore(ctx, ctx.retVal, ent, 1)
}

// Args: left, store?
// Returns: left - 1
// Stores: left <= left - 1
func vmOpDecrement(ctx *execContext, ent Entity) *Error {
	var (
		left uint64
		err  *Error
	)

	if left, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	// The result is stored back into the left operand
	ctx.retVal = left - 1
	if err := vmStore(ctx, ctx.retVal, ent.getArgs()[0]); err != nil {
		return err
	}
	return vmCondStore(ctx, ctx.retVal, ent, 1)
}

// Args: left, right, store?
// Returns: left * right
func vmOpMultiply(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left * right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args: left, right, remainder_store?, quotient_store?
// Returns: left / right; errDivideByZero if right == 0
func vmOpDivide(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	if right == 0 {
		return errDivideByZero
	}

	ctx.retVal = left / right
	if err = vmCondStore(ctx, ctx.retVal, ent, 3); err != nil {
		return err
	}

	// opDivide can also spefify a target for storing the remainder
	return vmCondStore(ctx, left%right, ent, 2)
}

// Args: left, right, remainder_store?
// Returns: left % right; errDivideByZero if right == 0
func vmOpMod(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	if right == 0 {
		return errDivideByZero
	}

	ctx.retVal = left % right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

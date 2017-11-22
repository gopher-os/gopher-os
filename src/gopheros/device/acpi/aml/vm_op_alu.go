package aml

import "bytes"

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

// Args; left, right, store?
// Returns left << right
func vmOpShiftLeft(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left << right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, right, store?
// Returns left >> right
func vmOpShiftRight(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left >> right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, right, store?
// Returns left & right
func vmOpBitwiseAnd(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left & right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, right, store?
// Returns left | right
func vmOpBitwiseOr(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left | right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, right, store?
// Returns left &^ right
func vmOpBitwiseNand(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left &^ right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, right, store?
// Returns ^(left | right)
func vmOpBitwiseNor(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = ^(left | right)
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, right, store?
// Returns left ^ right
func vmOpBitwiseXor(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left ^ right
	return vmCondStore(ctx, ctx.retVal, ent, 2)
}

// Args; left, store?
// Returns ^left
func vmOpBitwiseNot(ctx *execContext, ent Entity) *Error {
	var (
		left uint64
		err  *Error
	)

	if left, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	ctx.retVal = ^left
	return vmCondStore(ctx, ctx.retVal, ent, 1)
}

// Args; left, store?
// Returns the one-based bit location of the first MSb (most
// significant set bit). The result of 0 means no bit was
// set, 1 means the left-most bit set is the first bit, 2
// means the left-most bit set is the second bit, and so on.
func vmOpFindSetLeftBit(ctx *execContext, ent Entity) *Error {
	var (
		left uint64
		err  *Error
	)

	if left, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	ctx.retVal = uint64(0)
	for off, mask := uint8(1), uint64(1<<63); mask > 0; off, mask = off+1, mask>>1 {
		if left&mask != 0 {
			ctx.retVal = uint64(off)
			break
		}
	}

	return vmCondStore(ctx, ctx.retVal, ent, 1)
}

// Args; left, store?
// Returns the one-based bit location of the first LSb (least significant set
// bit). The result of 0 means no bit was set, 32 means the first bit set is
// the thirty-second bit, 31 means the first bit set is the thirty-first bit,
// and so on.
func vmOpFindSetRightBit(ctx *execContext, ent Entity) *Error {
	var (
		left uint64
		err  *Error
	)

	if left, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	ctx.retVal = uint64(0)
	for off, mask := uint8(1), uint64(1); off <= 64; off, mask = off+1, mask<<1 {
		if left&mask != 0 {
			ctx.retVal = uint64(off)
			break
		}
	}

	return vmCondStore(ctx, ctx.retVal, ent, 1)
}

// Args: left
// Returns !left
func vmOpLogicalNot(ctx *execContext, ent Entity) *Error {
	var (
		left uint64
		err  *Error
	)

	if left, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	ctx.retVal = left == 0
	return nil
}

// Args: left, right
// Returns left || right
func vmOpLogicalOr(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left != 0 || right != 0
	return nil
}

// Args: left, right
// Returns left && right
func vmOpLogicalAnd(ctx *execContext, ent Entity) *Error {
	var (
		left, right uint64
		err         *Error
	)

	if left, right, err = vmToIntArgs2(ctx, ent, 0, 1); err != nil {
		return err
	}

	ctx.retVal = left != 0 && right != 0
	return nil
}

// Args: left, right
// Returns left == right
// Operands must evaluate to either a Number, a String or a Buffer. The type
// of the first operand dictates the type of the second
func vmOpLogicalEqual(ctx *execContext, ent Entity) *Error {
	var (
		left, right interface{}
		argType     valueType
		err         *Error
		args        = ent.getArgs()
	)

	if len(args) != 2 {
		return errArgIndexOutOfBounds
	}

	if left, err = vmLoad(ctx, args[0]); err != nil {
		return err
	}

	if right, err = vmLoad(ctx, args[1]); err != nil {
		return err
	}

	argType = vmTypeOf(ctx, args[0])

	// Right operand must be coerced to the same type as left
	if right, err = vmConvert(ctx, right, argType); err != nil {
		return err
	}

	switch argType {
	case valueTypeInteger, valueTypeString:
		ctx.retVal = left == right
	case valueTypeBuffer:
		ctx.retVal = cmpBuffers(left.([]byte), right.([]byte)) == 0
	default:
		return errInvalidComparisonType
	}

	return nil
}

// Args: left, right
// Returns left < right
// Operands must evaluate to either a Number, a String or a Buffer. The type
// of the first operand dictates the type of the second
func vmOpLogicalLess(ctx *execContext, ent Entity) *Error {
	var (
		left, right interface{}
		argType     valueType
		err         *Error
		args        = ent.getArgs()
	)

	if len(args) != 2 {
		return errArgIndexOutOfBounds
	}

	if left, err = vmLoad(ctx, args[0]); err != nil {
		return err
	}

	if right, err = vmLoad(ctx, args[1]); err != nil {
		return err
	}

	argType = vmTypeOf(ctx, args[0])

	// Right operand must be coerced to the same type as left
	if right, err = vmConvert(ctx, right, argType); err != nil {
		return err
	}

	switch argType {
	case valueTypeInteger:
		ctx.retVal = left.(uint64) < right.(uint64)
	case valueTypeString:
		ctx.retVal = left.(string) < right.(string)
	case valueTypeBuffer:
		ctx.retVal = cmpBuffers(left.([]byte), right.([]byte)) == -1
	default:
		return errInvalidComparisonType
	}
	return err
}

// Args: left, right
// Returns left > right
// Operands must evaluate to either a Number, a String or a Buffer. The type
// of the first operand dictates the type of the second
func vmOpLogicalGreater(ctx *execContext, ent Entity) *Error {
	var (
		left, right interface{}
		argType     valueType
		err         *Error
		args        = ent.getArgs()
	)

	if len(args) != 2 {
		return errArgIndexOutOfBounds
	}

	if left, err = vmLoad(ctx, args[0]); err != nil {
		return err
	}

	if right, err = vmLoad(ctx, args[1]); err != nil {
		return err
	}

	argType = vmTypeOf(ctx, args[0])

	// Right operand must be coerced to the same type as left
	if right, err = vmConvert(ctx, right, argType); err != nil {
		return err
	}

	switch argType {
	case valueTypeInteger:
		ctx.retVal = left.(uint64) > right.(uint64)
	case valueTypeString:
		ctx.retVal = left.(string) > right.(string)
	case valueTypeBuffer:
		ctx.retVal = cmpBuffers(left.([]byte), right.([]byte)) == 1
	default:
		return errInvalidComparisonType
	}
	return err
}

// cmpBuffers compares left and right and returns 0 if they are equal, -1 if
// left < right and 1 if left > right. According to the ACPI spec, cmpBuffers
// will first compare the lengths of the slices before delegating a
// lexicographical comparison to bytes.Compare.
func cmpBuffers(left, right []byte) int {
	llen, rlen := len(left), len(right)
	if llen < rlen {
		return -1
	} else if llen > rlen {
		return 1
	}

	return bytes.Compare(left, right)
}

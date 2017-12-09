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
			case op == opBuffer:
				return typ.(*bufferEntity).data, nil
			default:
				// Val may be a nested opcode (e.g Add(Add(1,1), 2))
				// In this case, try evaluating the opcode and replace arg with the
				// output value that gets stored stored into ctx.retVal
				if err := ctx.vm.jumpTable[typ.getOpcode()](ctx, typ); err != nil {
					return nil, err
				}

				arg = ctx.retVal
			}
		case bool:
			// Convert boolean results to ints so they can be used
			// by the ALU comparators.
			if typ {
				return uint64(1), nil
			}
			return uint64(0), nil
		case *objRef:
			// According to p. 884 of the spec, reading from a
			// method argument reference automatically dereferences
			// the value
			if typ.isArgRef {
				return typ.ref, nil
			}

			// In all other cases we return back the reference itself
			return typ, nil
		default:
			return typ, nil
		}
	}
}

// vmCondStore is a wrapper around vmWrite that checks whether argIndex
// contains a non-nil target before attempting to write val to it. If argIndex
// is out of bounds or it points to a nil target then this function behaves as
// a no-op.
func vmCondStore(ctx *execContext, val interface{}, ent Entity, argIndex int) *Error {
	args := ent.getArgs()
	if len(args) <= argIndex || vmIsNilTarget(args[argIndex]) {
		return nil
	}

	return vmStore(ctx, val, args[argIndex])
}

// vmStore attempts to write the value contained in src to dst.
func vmStore(ctx *execContext, src, dst interface{}) *Error {
	if dst == nil || src == nil {
		return errNilStoreOperands
	}

	// The target should be some type of AML Entity
	dstEnt, ok := dst.(Entity)
	if !ok {
		return errInvalidStoreDestination
	}
	dstOp := dstEnt.getOpcode()

	// According to the spec, storing to a constant is a no-op and not a
	// fatal error. In addition, if the destination is the Debug opbject,
	// the interpreter must display the value written to it. This
	// interpreter implementation just treats this as a no-op.
	if _, ok := dst.(*constEntity); ok || dstOp == opDebug {
		return nil
	}

	// The spec requires the interpreter to make a copy of the src object
	// and apply the appropriate conversions depending on the destination
	// object type
	srcCopy, err := vmCopyObject(ctx, src)
	if err != nil {
		return err
	}

	switch {
	case opIsLocalArg(dstOp):
		// According to p.897 of the spec, writing to a local object
		// always overwrites the previous value with a copy of src even
		// if this is an object reference
		ctx.localArg[dstOp-opLocal0] = srcCopy
		return nil
	case opIsMethodArg(dstOp):
		// According to p.896 of the spec, if ArgX is a reference
		// we need to dereference it and store the copied object
		// in the reference. In all other cases we just overwrite the
		// value in ArgX with the object copy
		if dstRef, isRef := ctx.methodArg[dstOp-opArg0].(*objRef); isRef {
			dstRef.ref = srcCopy
		} else {
			ctx.methodArg[dstOp-opArg0] = srcCopy
		}

		return nil
	}

	return &Error{message: "vmStore: unsupported opcode: " + dstOp.String()}
}

// vmIsNilTarget returns true if t is nil or a nil const entity.
func vmIsNilTarget(target interface{}) bool {
	if target == nil {
		return true
	}

	if ent, ok := target.(*constEntity); ok {
		return ent.val != nil && ent.val != 0
	}

	return false
}

// vmCopyObject returns a copy of obj.
func vmCopyObject(ctx *execContext, obj interface{}) (interface{}, *Error) {
	switch typ := obj.(type) {
	case string:
		return typ, nil
	case uint64:
		return typ, nil
	}
	return nil, errCopyFailed
}

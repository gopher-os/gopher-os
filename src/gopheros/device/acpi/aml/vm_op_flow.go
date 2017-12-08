package aml

import (
	"bytes"
	"gopheros/kernel/kfmt"
)

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

func vmOpBreak(ctx *execContext, ent Entity) *Error {
	ctx.ctrlFlow = ctrlFlowTypeBreak
	return nil
}

func vmOpContinue(ctx *execContext, ent Entity) *Error {
	ctx.ctrlFlow = ctrlFlowTypeContinue
	return nil
}

// Args: Predicate {TermList}
// Execute the scoped termlist block until predicate evaluates to false or any
// of the instructions in the TermList changes the control flow to break or
// return.
func vmOpWhile(ctx *execContext, ent Entity) *Error {
	var (
		predRes     interface{}
		err         *Error
		whileBlock  ScopeEntity
		isScopedEnt bool
		args        = ent.getArgs()
		argLen      = len(args)
	)

	if argLen != 2 {
		return errArgIndexOutOfBounds
	}

	if whileBlock, isScopedEnt = args[1].(ScopeEntity); !isScopedEnt {
		return errWhileBodyNotScopedEntity
	}

	for err == nil {
		if predRes, err = vmLoad(ctx, args[0]); err != nil {
			continue
		}

		if predResAsUint, isUint := predRes.(uint64); !isUint || predResAsUint != 1 {
			break
		}

		err = execBlock(ctx, whileBlock)
		if ctx.ctrlFlow == ctrlFlowTypeFnReturn {
			// Preserve return flow type so we exit the innermost function
			break
		} else if ctx.ctrlFlow == ctrlFlowTypeBreak {
			// Exit while block and switch to sequential execution for the code
			// that follows. The current IP needs to be adjusted to point to the
			// end of the current block
			ctx.IP = whileBlock.blockEndIPOffset()
			ctx.ctrlFlow = ctrlFlowTypeNextOpcode
			break
		}

		// Restart while block but reset to sequential execution so the predicate
		// and while body can be properly evaluated
		ctx.ctrlFlow = ctrlFlowTypeNextOpcode
	}

	return err
}

// Args: Predicate {Pred == true TermList} {Pref == false TermList}?
//
// Execute the scoped term list if predicate evaluates to true; If predicate
// evaluates to false and the optional else block is defined then it will be
// executed instead.
func vmOpIf(ctx *execContext, ent Entity) *Error {
	var (
		predRes            interface{}
		err                *Error
		ifBlock, elseBlock ScopeEntity
		isScopedEnt        bool
		args               = ent.getArgs()
		argLen             = len(args)
	)

	if argLen < 2 || argLen > 3 {
		return errArgIndexOutOfBounds
	}

	if ifBlock, isScopedEnt = args[1].(ScopeEntity); !isScopedEnt {
		return errIfBodyNotScopedEntity
	}

	// Check for the optional else block
	if argLen == 3 {
		if elseBlock, isScopedEnt = args[2].(ScopeEntity); !isScopedEnt {
			return errElseBodyNotScopedEntity
		}
	}

	if predRes, err = vmLoad(ctx, args[0]); err != nil {
		return err
	}

	if predResAsUint, isUint := predRes.(uint64); !isUint || predResAsUint == 1 {
		return execBlock(ctx, ifBlock)
	} else if elseBlock != nil {
		return execBlock(ctx, elseBlock)
	}

	return nil
}

// vmOpMethodInvocation dispatches a method invocation and sets ctx.retVal
// to the value returned by the method invocation. This function also supports
// invocation of methods that are provided by the kernel host such as the ones
// defined in section 5.7 of the ACPI spec.
func vmOpMethodInvocation(ctx *execContext, ent Entity) *Error {
	// Make sure the target method is properly resolved
	inv := ent.(*methodInvocationEntity)
	if !inv.Resolve(ctx.vm.errWriter, ctx.vm.rootNS) {
		return &Error{message: "call to undefined method: " + inv.methodName}
	}

	return ctx.vm.execMethod(ctx, inv.method, ent.getArgs()...)
}

// Args: type, code, arg
//
// Generate an OEM-defined fatal error. The OSPM must catch this error,
// optionally log it and perform a controlled system shutdown
func vmOpFatal(ctx *execContext, ent Entity) *Error {
	var (
		buf     bytes.Buffer
		errType uint64
		errCode uint64
		errArg  uint64
		err     *Error
	)

	if errType, err = vmToIntArg(ctx, ent, 0); err != nil {
		return err
	}

	if errCode, errArg, err = vmToIntArgs2(ctx, ent, 1, 2); err != nil {
		return err
	}

	kfmt.Fprintf(&buf, "fatal OEM-defined error (type: 0x%x, code: 0x%x, arg: 0x%x)", errType, errCode, errArg)
	return &Error{message: buf.String()}
}

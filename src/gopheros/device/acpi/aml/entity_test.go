package aml

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func TestEntityMethods(t *testing.T) {
	specs := []Entity{
		&unnamedEntity{},
		&constEntity{},
		&scopeEntity{},
		&bufferEntity{},
		&fieldUnitEntity{},
		&indexFieldEntity{},
		&namedReference{},
		&methodInvocationEntity{},
		&Method{},
		&Device{},
		&mutexEntity{},
		&eventEntity{},
	}

	t.Run("table handle methods", func(t *testing.T) {
		exp := uint8(42)
		for specIndex, spec := range specs {
			spec.setTableHandle(exp)
			if got := spec.TableHandle(); got != exp {
				t.Errorf("[spec %d] expected to get back handle %d; got %d", specIndex, exp, got)
			}
		}
	})

	t.Run("append/remove/get parent methods", func(t *testing.T) {
		parent := &scopeEntity{name: `\`}

		for specIndex, spec := range specs {
			parent.Append(spec)
			if got := spec.Parent(); got != parent {
				t.Errorf("[spec %d] expected to get back parent %v; got %v", specIndex, parent, got)
			}

			parent.removeChild(spec)
		}

		if got := len(parent.Children()); got != 0 {
			t.Fatalf("expected parent not to have any child nodes; got %d", got)
		}
	})
}

func TestEntityArgAssignment(t *testing.T) {
	specs := []struct {
		ent         Entity
		argList     []interface{}
		expArgList  []interface{}
		limitedArgs bool
	}{
		{
			&unnamedEntity{},
			[]interface{}{"foo", 1, "bar"},
			[]interface{}{"foo", 1, "bar"},
			false,
		},
		{
			&constEntity{},
			[]interface{}{"foo"},
			nil, // constEntity populates its internal state using the 1st arg
			true,
		},
		{
			&scopeEntity{},
			[]interface{}{"foo", 1, 2, 3},
			[]interface{}{1, 2, 3}, // scopeEntity will treat arg0 as the scope name if it is a string
			false,
		},
		{
			&bufferEntity{},
			[]interface{}{1, []byte{}},
			nil, // bufferEntity populates its internal state using the first 2 args
			true,
		},
		{
			&regionEntity{},
			[]interface{}{"REG0", uint64(0x4), 0, 10},
			[]interface{}{0, 10}, // region populates its internal state using the first 2 args
			true,
		},
		{
			&mutexEntity{},
			[]interface{}{"MUT0", uint64(1)},
			nil, // mutexEntity populates its internal state using the first 2 args
			true,
		},
	}

nextSpec:
	for specIndex, spec := range specs {
		for i, arg := range spec.argList {
			if !spec.ent.setArg(uint8(i), arg) {
				t.Errorf("[spec %d] error setting arg %d", specIndex, i)
				continue nextSpec
			}
		}

		if spec.limitedArgs {
			if spec.ent.setArg(uint8(len(spec.argList)), nil) {
				t.Errorf("[spec %d] expected additional calls to setArg to return false", specIndex)
				continue nextSpec
			}
		}

		if got := spec.ent.getArgs(); !reflect.DeepEqual(got, spec.expArgList) {
			t.Errorf("[spec %d] expected to get back arg list %v; got %v", specIndex, spec.expArgList, got)
		}
	}
}

func TestEntityResolveErrors(t *testing.T) {
	scope := &scopeEntity{name: `\`}

	specs := []resolver{
		// Unknown connection entity
		&fieldUnitEntity{connectionName: "CON0"},
		// Unknown region
		&fieldUnitEntity{connectionName: `\`, regionName: "REG0"},
		// Unknown connection entity
		&indexFieldEntity{connectionName: "CON0"},
		// Unknown index register
		&indexFieldEntity{connectionName: `\`, indexRegName: "IND0"},
		// Unknown data register
		&indexFieldEntity{connectionName: `\`, indexRegName: `\`, dataRegName: "DAT0"},
		// Unknown reference
		&namedReference{unnamedEntity: unnamedEntity{parent: scope}, targetName: "TRG0"},
	}

	for specIndex, spec := range specs {
		if spec.Resolve(ioutil.Discard, scope) {
			t.Errorf("[spec %d] expected Resolve() to fail", specIndex)
		}
	}
}

package service

import (
	"errors"
	"fmt"
	"reflect"
)

type container struct {
	parent   *container
	objTable map[reflect.Type]reflect.Value
}

func newContainer(p *container) *container {
	return &container{
		parent:   p,
		objTable: map[reflect.Type]reflect.Value{},
	}
}

func (c *container) invokeAndProcess(ctr interface{}, resFunc func([]reflect.Value) error) error {
	// Check for function type
	f := reflect.TypeOf(ctr)
	if e := checkFunc(ctr, f); e != nil {
		return e
	}

	// Build the arguments list
	args, err := c.buildArgs(f)
	if err != nil {
		return err
	}

	// Call the function
	returned := reflect.ValueOf(ctr).Call(args)

	// process results
	return resFunc(returned)
}

func (c *container) invoke(ctr interface{}) error {
	return c.invokeAndProcess(ctr, func(returned []reflect.Value) error {
		return checkError(returned)
	})
}

func (c *container) addWithProcessValue(ctr interface{}, vf func(v reflect.Value)) error {
	return c.invokeAndProcess(ctr, func(returned []reflect.Value) error {
		if e := checkError(returned); e != nil {
			return e
		}

		for _, v := range returned {
			if e := c.put(v); e != nil {
				return e
			}
			if vf != nil {
				// Call the value processor
				vf(v)
			}
		}
		return nil
	})
}

func (c *container) add(ctr interface{}) error {
	return c.addWithProcessValue(ctr, nil)
}

func (c *container) buildArgs(ctrType reflect.Type) ([]reflect.Value, error) {
	numArgs := ctrType.NumIn()
	if ctrType.IsVariadic() {
		// Ignore the variadic argument
		numArgs--
	}
	vals := make([]reflect.Value, 0, numArgs)
	for i := 0; i < numArgs; i++ {
		v, err := c.get(ctrType.In(i))
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func (c *container) get(in reflect.Type) (reflect.Value, error) {
	// Always find the value in the parent type first.
	if c.parent != nil {
		v, err := c.parent.get(in)

		// We found the value in our ancestry, so return that value.
		if err == nil {
			return v, err
		}
	}

	// Check in this container for the value
	inType := in
	if in.Kind() == reflect.Ptr {
		inType = in.Elem()
	}
	v, ok := c.objTable[inType]

	if !ok {
		return v, fmt.Errorf("dependency for type %v not found", in)
	}

	// Found Value!
	return v, nil
}

func (c *container) put(v reflect.Value) error {
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if _, err := c.get(t); err == nil {
		return fmt.Errorf("type %v is already present", v.Type())
	}
	c.objTable[t] = v
	return nil
}

var (
	_errType = reflect.TypeOf((*error)(nil)).Elem()
)

func checkError(returned []reflect.Value) error {
	if len(returned) == 0 {
		return nil
	}
	if last := returned[len(returned)-1]; last.Type().Implements(_errType) {
		if err, _ := last.Interface().(error); err != nil {
			return err
		}
	}
	return nil
}

func checkFunc(ctr interface{}, f reflect.Type) error {
	if f == nil {
		return errors.New("can't invoke an untyped nil")
	}
	if f.Kind() != reflect.Func {
		return fmt.Errorf("can't invoke non-function %v (type %v)", ctr, f)
	}
	return nil
}

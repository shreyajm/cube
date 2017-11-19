package service

import (
	"errors"
	"fmt"
	"reflect"
)

// Container provides dependency injection for services. Each container keeps
// track of an object table that maps a type to its instantiation. As a new
// service constructor is added, the container checks its parameters and
// marks them as dependencies and caches the return value in the object table.
// Each dependency is evaluated as soon as the constructor is added to the
// container.
//
// Containers are chained with parent first delegation model. This means that
// when ever a dependency is evaluated, the container checks if its parent
// container provides the object first before checking its own object table.
//
// By adding the services in their order of dependency into a container and
// by chaining these containers we can build the complete static dependency
// graph of a process.
type container struct {
	parent   *container
	objTable map[reflect.Type]reflect.Value
}

// newContainer creates a new container with a parent container, if parent
// is nil it is a root container.
func newContainer(p *container) *container {
	return &container{
		parent:   p,
		objTable: map[reflect.Type]reflect.Value{},
	}
}

// resultProcessor is a function handler that can process the
// results produced by a constructor.
type resultProcessor func([]reflect.Value) error

// invokeAndProcess invokes a provided constructor by resolving its dependencies
// and calls the result processor on the results if the constructor succeeds.
func (c *container) invokeAndProcess(ctr interface{}, resFunc resultProcessor) error {
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

	// Check for errors
	if err := checkError(returned); err != nil {
		return err
	}

	// Process results if a processor is provided
	if resFunc != nil {
		return resFunc(returned)
	}
	return nil
}

// invoke call the constructor provided.
func (c *container) invoke(ctr interface{}) error {
	return c.invokeAndProcess(ctr, nil)
}

// value processor is handler that can process each value produced
// while adding a service constructor. Handlers can be used to cache
// object instances externally for lifecycle invocations.
type valueProcessor func(reflect.Value)

// add invokes the constructor and calls the value processor on each
// object the constructor produced.
func (c *container) add(ctr interface{}, vf valueProcessor) error {
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

// buildArgs builds the arguments required by the constructor by looking
// up the object table.
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

// get finds a object required by buildArgs. It looks up the parent
// container first for the object and then the object table of this
// container.
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

// put caches the object produced in the object table of the container.
// If the object is already present in the object table, the value is
// rejected. This makes sure that there is only one constructor that
// can produce and object of a specific type in a given container hierarchy.
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

// checkError checks if the value list ends with an error type and returns
// that error. this is used to see if a constructor produced an error.
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

// checkFunc checks if a constructor is a valid function that can be invoked.
func checkFunc(ctr interface{}, f reflect.Type) error {
	if f == nil {
		return errors.New("can't invoke an untyped nil")
	}
	if f.Kind() != reflect.Func {
		return fmt.Errorf("can't invoke non-function %v (type %v)", ctr, f)
	}
	return nil
}

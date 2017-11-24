package di

import (
	"errors"
	"fmt"
	"reflect"
)

// Container provides dependency injection for components. Each container keeps
// track of an object table that maps a type to its object. As a new
// component constructor is added, the container checks its parameters and
// marks them as dependencies and caches the return value in the object table.
// Each dependency is evaluated as soon as the constructor is added to the
// container.
//
// Containers are chained with parent first delegation model. This means that
// when ever a dependency is evaluated, the container checks if its parent
// container provides the object first before checking its own object table.
//
// By adding the components in their order of dependency into a container and
// by chaining these containers we can build the complete static dependency
// graph of a process.
type Container struct {
	parent   *Container
	objTable map[reflect.Type]reflect.Value
	dups     []reflect.Type
}

// New creates a new container chained to a parent container, if parent
// is nil it is a root container.
func New(p *Container, dups ...reflect.Type) *Container {
	return &Container{
		parent:   p,
		objTable: map[reflect.Type]reflect.Value{},
		dups:     dups,
	}
}

// ValueProcessor is handler that can process each value produced while executing
// a function using the dependency container. Handlers can be used to cache object
// instances externally for lifecycle invocations.
type ValueProcessor func(reflect.Value) error

// Invoke a function evaluating its dependencies using this container. If the
// invoked function returns an error Invoke will return that error to the caller.
// If a value processor is provided, each value returned by the invoked function
// is processed using the value processor. If the value processor returns an
// error, that error is returned to the caller of Invoke.
//
// Note the any return values from the invoked function are not cached the container.
func (c *Container) Invoke(fx interface{}, vp ValueProcessor) error {
	// Check for function type
	f := reflect.TypeOf(fx)
	if e := checkFunc(fx, f); e != nil {
		return e
	}

	// Build the arguments list
	args, err := c.buildArgs(f)
	if err != nil {
		return err
	}

	// Call the function
	returned := reflect.ValueOf(fx).Call(args)

	// Check for errors
	if err := checkError(returned); err != nil {
		return err
	}

	// Process results if a processor is provided
	if vp != nil {
		for _, v := range returned {
			if err := vp(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Add invokes the provided constructor evaluating its dependencies using this container.
// If the constructor returns an error Add will return that error to the caller and rejects
// any values produced by the constructor. If success, the values returned by the constructor
// are cached in this container against their types and can be used for subsequent
// dependency calculations.
//
// If the type of the value is produced by the constructor is already present in this
// container or its ancestors, the value is rejected and Add returns an error.
//
// If a value processor is provided, Add calls the value processor function on all returned
// values of the constructor. This can used to cache the values outside the container.
func (c *Container) Add(ctr interface{}, vp ValueProcessor) error {
	vals := []reflect.Value{}
	resProc := func(v reflect.Value) error {
		t := valType(v)
		if _, err := c.get(t); err == nil {
			return fmt.Errorf("type %v is already present", v.Type())
		}
		if vp != nil {
			// Call the value processor passed by the caller of Add
			if err := vp(v); err != nil {
				return err
			}
		}
		vals = append(vals, v)
		return nil
	}

	// Invoke this constructor with our own result processor
	if err := c.Invoke(ctr, resProc); err != nil {
		return err
	}

	// Cache all the values produced by this invocation.
	for _, v := range vals {
		t := valType(v)
		c.objTable[t] = v
	}
	return nil
}

// buildArgs builds the arguments required by the constructor by looking
// up the object table.
func (c *Container) buildArgs(ctrType reflect.Type) ([]reflect.Value, error) {
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

func (c *Container) checkParent(in reflect.Type) bool {
	if c.parent != nil {
		for _, dup := range c.dups {
			if in == dup {
				return false
			}
		}
		return true
	}
	return false
}

// get finds a object required by buildArgs. It looks up the parent
// container first for the object and then the object table of this
// container.
func (c *Container) get(in reflect.Type) (reflect.Value, error) {
	// Always find the value in the parent type first.
	if c.checkParent(in) {
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

// Returns the type of this value
func valType(v reflect.Value) reflect.Type {
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

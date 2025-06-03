package tpp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// newReflectedMockCall returns an instrumented MockCall by using reflect.
//
// We need this because we're interested in functions such as "Return" on the
// mock, but Mockery types this depending on the thing being mocked and Go's
// generics aren't rich enough for us to define a generic "Return" on the
// MockCall interface.
//
// We *only* use reflection for that reason. All of the functions being accessed
// here are exported functions from the Mockery mock. We must never touch
// anything unexported here, and perhaps one day this layer can be removed.
func newReflectedMockCall(mock MockCall) (*reflectedMockCall, error) {
	// Validate mock
	mockval := reflect.ValueOf(mock)

	if mockval.Kind() == reflect.Ptr {
		mockval = mockval.Elem()
	}

	if mockval.Kind() != reflect.Struct {
		return nil, errors.New("mock must be struct")
	}

	// Extract and validate Arguments
	args := mockval.FieldByName("Arguments")

	if !args.IsValid() {
		return nil, errors.New("args must be valid")
	}

	if args.Kind() != reflect.Slice {
		return nil, errors.New("args must be slice")
	}

	if !args.CanSet() {
		return nil, errors.New("args must be mutable")
	}

	// Extract and validate Return
	ret := reflect.ValueOf(mock).MethodByName("Return")
	if !ret.IsValid() {
		return nil, errors.New("given mock has no Return method")
	}

	return &reflectedMockCall{
		wrapped:      mock,
		args:         args,
		returnMethod: ret,
	}, nil
}

type reflectedMockCall struct {
	wrapped      MockCall
	args         reflect.Value
	returnMethod reflect.Value
}

// GetArguents returns the mock's arguments.
//
// This is just a getter for mock.Arguments. We need this because we want to get
// the arguments but an interface like tpp.MockCall can't specify field values
// in Golang.
func (rm *reflectedMockCall) GetArguments() ([]any, error) {
	result := make([]any, rm.args.Len())

	for i := 0; i < rm.args.Len(); i++ {
		result[i] = rm.args.Index(i).Interface()
	}

	return result, nil
}

// GetArguents returns the mock's arguments.
//
// This is just a setter for mock.Arguments. We need this because we want to set
// the arguments but an interface like tpp.MockCall can't specify field values
// in Golang.
func (rm *reflectedMockCall) SetArguments(args []any) {
	newSlice := reflect.MakeSlice(rm.args.Type(), len(args), len(args))

	for i, a := range args {
		newSlice.Index(i).Set(reflect.ValueOf(a))
	}

	rm.args.Set(newSlice)
}

// CallReturnEmpty calls the mock's Return method with empty values.
//
// If an optional error is provided, we will use that for error values.
func (rm *reflectedMockCall) CallReturnEmpty(retErr error) {
	var (
		returnType = rm.returnMethod.Type()
		returnLen  = returnType.NumIn()
		emptyArgs  = make([]reflect.Value, 0)
	)

	if returnLen == 1 && returnType.In(0).Name() == "" && retErr != nil {
		// Special case: we have an error to return and one argument with an unknown
		// type. This can happen in two cases. First, when we're handling a bare
		// testify mock and don't know the Return type. Secondly, when we're handling
		// a mockery mock with a custom Return type which will be hidden to us. In
		// the first case, we want to call Return(retErr). The second case is a user
		// error and will panic.
		emptyArgs = append(emptyArgs, reflect.ValueOf(retErr))
	} else {
		for i := 0; i < returnLen; i++ {
			argType := returnType.In(i)

			if retErr != nil && argType.Name() == "error" {
				// We were given an error to return -- use it!
				emptyArgs = append(emptyArgs, reflect.ValueOf(retErr))
			} else if !returnType.IsVariadic() || i < returnLen-1 {
				emptyArgs = append(emptyArgs, reflect.Zero(argType))
			}
		}
	}

	rm.mustArgMatch(returnType, emptyArgs)

	rm.returnMethod.Call(emptyArgs)
}

// CallReturn calls the mock's Return method with the given args.
//
// If an optional retErr is provided, we will use that for error values.
func (rm *reflectedMockCall) CallReturn(args []any, retErr error) error {
	var (
		returnType = rm.returnMethod.Type()
		returnLen  = returnType.NumIn()
		returnArgs = append([]any{}, args...)
	)

	if retErr != nil {
		// We were given an error to return -- use it!
		returnArgs = append(returnArgs, retErr)
	} else {
		// Add a nil error, if applicable
		for i := len(args); i < returnLen; i++ {
			if returnType.In(i).Name() == "error" {
				var emptyErr error
				returnArgs = append(returnArgs, emptyErr)
				break
			}
		}
	}

	rargs, err := toReflectValues(returnArgs, returnType)
	if err != nil {
		return fmt.Errorf("toReflectValues failed to transform return values: %s", err)
	}

	rm.mustArgMatch(returnType, rargs)

	rm.returnMethod.Call(rargs)
	return nil
}

// toReflectValues transforms the |args| of the |method| from `[]any` to
// `[]reflect.Value`.
func toReflectValues(args []any, typ reflect.Type) ([]reflect.Value, error) {
	if len(args) != typ.NumIn() && !typ.IsVariadic() {
		return nil, fmt.Errorf(
			"mismatched number of args: expected %d but got %d",
			typ.NumIn(),
			len(args),
		)
	}

	values := make([]reflect.Value, len(args))

	for i, arg := range args {
		argType := typ.In(min(i, typ.NumIn()-1))

		if arg != nil {
			values[i] = reflect.ValueOf(arg)
		} else {
			switch argType.Kind() {
			case reflect.Interface, reflect.Ptr:
				values[i] = reflect.Zero(argType)
			default:
				return nil, fmt.Errorf(
					"cannot handle nil for non-interface or non-pointer type: %s",
					argType,
				)
			}
		}
	}

	return values, nil
}

// -----------------------------------------------------------------------------
// Helpful error messages ------------------------------------------------------
// -----------------------------------------------------------------------------
// Because of lack of type safety, people are going to both:
//   (a) pass in the wrong number of arguments/returns, and
//   (b) pass in the wrong type of arguments/returns
// especially when refactoring code. We need to make sure that the errors one
// gets back in these two cases are as helpful as possible.
// -----------------------------------------------------------------------------

// mustArgMatch panics with a helpful message if the args don't match the type.
func (rm *reflectedMockCall) mustArgMatch(fnType reflect.Type, args []reflect.Value) {
	if !argsMatch(fnType, args) {
		panic(printArgMismatch(reflect.ValueOf(rm.wrapped).String(), fnType, args))
	}
}

// argsMatch returns whether the args match the given function type.
func argsMatch(fnType reflect.Type, args []reflect.Value) bool {
	if fnType.Kind() != reflect.Func {
		return false
	}

	numIn := fnType.NumIn()

	// Handle variadic function separately
	if fnType.IsVariadic() {
		if len(args) < numIn-1 {
			return false
		}
		for i := 0; i < numIn-1; i++ {
			if !argAssignable(args[i], fnType.In(i)) {
				return false
			}
		}
		variadicType := fnType.In(numIn - 1).Elem()
		for i := numIn - 1; i < len(args); i++ {
			if !argAssignable(args[i], variadicType) {
				return false
			}
		}
		return true
	}

	// Non-variadic function
	if len(args) != numIn {
		return false
	}
	for i := 0; i < numIn; i++ {
		if !argAssignable(args[i], fnType.In(i)) {
			return false
		}
	}
	return true
}

func argAssignable(arg reflect.Value, target reflect.Type) bool {
	if !arg.IsValid() {
		// Invalid value = nil; only assignable to nillable types
		kind := target.Kind()
		return kind == reflect.Interface ||
			kind == reflect.Ptr ||
			kind == reflect.Slice ||
			kind == reflect.Map ||
			kind == reflect.Func ||
			kind == reflect.Chan
	}
	return arg.Type().AssignableTo(target)
}

func printArgMismatch(debugName string, fnType reflect.Type, args []reflect.Value) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\nReturn() called with the wrong arguments!\n"))
	b.WriteString(fmt.Sprintf("    Function: %s\n", debugName))

	numIn := fnType.NumIn()

	// Expected signature
	b.WriteString("    Expected: (")
	for i := 0; i < numIn; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		if fnType.IsVariadic() && i == numIn-1 {
			b.WriteString("..." + fnType.In(i).Elem().String())
		} else {
			b.WriteString(fnType.In(i).String())
		}
	}
	b.WriteString(")\n")

	// Actual signature
	b.WriteString("    Received: (")
	for i := 0; i < len(args); i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		if !args[i].IsValid() {
			b.WriteString("invalid")
		} else {
			b.WriteString(args[i].Type().String())
		}
	}
	b.WriteString(")\n")
	b.WriteString("\n")

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

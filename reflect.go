package tpp

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

type reflectedMockCall struct {
	args         reflect.Value
	returnMethod reflect.Value
}

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
	//ret := mockval.MethodByName("Return")
	ret := reflect.ValueOf(mock).MethodByName("Return")
	if !ret.IsValid() {
		return nil, errors.New("given mock has no Return method")
	}

	return &reflectedMockCall{
		args:         args,
		returnMethod: ret,
	}, nil
}

func (rm *reflectedMockCall) GetArguments() ([]any, error) {
	result := make([]any, rm.args.Len())

	for i := 0; i < rm.args.Len(); i++ {
		result[i] = rm.args.Index(i).Interface()
	}

	return result, nil
}

func (rm *reflectedMockCall) SetArguments(args []any) {
	newSlice := reflect.MakeSlice(rm.args.Type(), len(args), len(args))

	for i, a := range args {
		newSlice.Index(i).Set(reflect.ValueOf(a))
	}

	rm.args.Set(newSlice)
}

func (rm *reflectedMockCall) CallReturnEmpty(retErr error) {
	var (
		returnType = rm.returnMethod.Type()
		returnLen  = returnType.NumIn()
		emptyArgs  = make([]reflect.Value, 0)
	)

	for i := 0; i < returnLen; i++ {
		argType := returnType.In(i)

		if retErr != nil && (argType.Name() == "error" || argType.Name() == "") {
			// We were given an error to return -- use it!
			// Note we match on "error" and "" here because if a bare testifymock.Call
			// has been passed to us, we won't have type information. We don't usually
			// add a return in that case (see below), but here the user has set an err
			// so we do what they ask.
			emptyArgs = append(emptyArgs, reflect.ValueOf(retErr))
		} else if argType.Name() != "" {
			emptyArgs = append(emptyArgs, reflect.Zero(argType))
		}
	}

	rm.returnMethod.Call(emptyArgs)
}

func (rm *reflectedMockCall) CallReturn(args []any, retErr error) error {
	returnType := rm.returnMethod.Type()
	returnLen := returnType.NumIn()

	returnArgs := append([]any{}, args...)

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

	// TODO: some check here that the correct number and type of
	// arguments have been passed in

	rargs, err := toReflectValues(returnArgs, returnType)
	if err != nil {
		return fmt.Errorf("toReflectValues failed to transform return values: %s", err)
	}

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
			case reflect.Interface:
				values[i] = reflect.Zero(argType)
			case reflect.Ptr:
				values[i] = reflect.New(argType.Elem()).Elem()
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package tpp

import (
	"fmt"
	"reflect"
	"testing"

	testifymock "github.com/stretchr/testify/mock"
)

// True returns a ptr to true. Tri-state bools, baby!
func True() *bool {
	v := true
	return &v
}

// False returns a ptr to false. Tri-state bools, baby!
func False() *bool {
	v := false
	return &v
}

// Call represents a single mock call. This is to be used with Expects.
type Call struct {
	Given  []any
	Return []any
}

// Expects represents an expectation for use in configuration-driven tests.
// This binds together whether a set of mock calls should be expected, what
// they should return, and whether they should return an error.
type Expects struct {
	Expected *bool
	Calls    []Call
	err      bool
}

// Unexpecteds returns an Expects which is unexpected.
func Unexpecteds() Expects {
	return Expects{
		Expected: ptr(false),
	}
}

// OKs returns an Expects with the given calls.
func OKs(calls []Call) Expects {
	return Expects{
		Expected: ptr(true),
		Calls:    calls,
	}
}

// Errs returns an Expects with a generic test error.
func Errs() Expects {
	return Expects{
		Expected: ptr(true),
		err:      true,
	}
}

// Expectorise configures the given |mockFunc| according to the behaviour
// specified in the Expects.
//
// If Expects.Expected is false, nothing will be done.
//
// If Expects.Expected is nil, Expectorise will configure the mock func to be
// Maybe() called with mock.Anything args and return the given |defaultReturns|.
// If an Expects.Error is also specified, an error will be returned.
//
// If Expects.Expected is true and Expects.Calls is non-empty, Expectorise will
// configure the mock func to be called with the Expect.Calls arguments and to
// return the Expect.Calls returns.
func (e *Expects) Expectorise(t *testing.T, mockFunc any, defaultReturns []any) {
	if e.Expected != nil && !*e.Expected {
		// Nothing is expected; nothing to set up.
		return
	}

	fn := reflect.ValueOf(mockFunc)
	if fn.Kind() != reflect.Func {
		t.Fatalf("mockFunc is not a func")
		return
	}

	if e.Expected == nil || e.Calls == nil || len(e.Calls) == 0 {
		// Set up a default call, injecting an error one is specified
		args := make([]any, 0, fn.Type().NumIn())
		for i := 0; i < fn.Type().NumIn(); i++ {
			args = append(args, testifymock.Anything)
		}
		err := configureMockCall(
			fn,
			args,
			defaultReturns,
			configureMockCallOpts{setReturnMaybe: true, injectErrReturn: e.err},
		)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	// Set up the provided calls
	for _, call := range e.Calls {
		err := configureMockCall(fn, call.Given, call.Return, configureMockCallOpts{})
		if err != nil {
			t.Fatal(err)
		}
	}
}

type configureMockCallOpts struct {
	setReturnMaybe  bool
	injectErrReturn bool
}

func configureMockCall(
	fn reflect.Value,
	args []any,
	returns []any,
	opts configureMockCallOpts,
) error {
	inputArgs, err := toReflectValuesDeprecated(args, fn)
	if err != nil {
		return fmt.Errorf("toReflectValues failed: %s", err.Error())
	}

	expectCall := fn.Call(inputArgs)
	if len(expectCall) == 0 {
		return fmt.Errorf("calling mockFunc did not return a valid Call object")
	}

	returnMethod := expectCall[0].MethodByName("Return")
	if !returnMethod.IsValid() {
		return fmt.Errorf("mock.Call does not have a Return method")
	}

	returnArgs := make([]reflect.Value, len(returns))
	for i, ret := range returns {
		if opts.injectErrReturn && returnMethod.Type().In(i).Name() == "error" {
			returnArgs[i] = reflect.ValueOf(errTest)
		} else if ret == nil {
			returnArgs[i] = reflect.Zero(returnMethod.Type().In(i))
		} else {
			returnArgs[i] = reflect.ValueOf(ret)
		}
	}

	returnMethod.Call(returnArgs)

	if opts.setReturnMaybe {
		maybeMethod := expectCall[0].MethodByName("Maybe")
		if !maybeMethod.IsValid() {
			return fmt.Errorf("mock.Call does not have a Maybe method")
		}
		maybeMethod.Call(nil)
	}
	return nil
}

// toReflectValuesDeprecated transforms the |args| of the |method| from `[]any` to
// `[]reflect.Value`.
func toReflectValuesDeprecated(args []any, method reflect.Value) ([]reflect.Value, error) {
	methodType := method.Type()

	if len(args) != methodType.NumIn() {
		return nil, fmt.Errorf(
			"mismatched number of args: expected %d but got %d",
			methodType.NumIn(),
			len(args),
		)
	}

	values := make([]reflect.Value, len(args))

	for i, arg := range args {
		argType := methodType.In(i)

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

// t++ (t plus plus)
//
// This package has some generic helpers to facilitate writing configuration
// driven tests. These are where one single meta-test is written which is then
// configured by a passed in struct, which defines the actual test behaviour.
package tpp

/*
Here's a template for writing such tests:

	func TestXXX(t *testing.T) {
		for _, tt := range []struct {
			name    string
			getFoo  tpp.Expect
			wantErr bool
		}{
			{
				name:    "OK",
				getFoo:  tpp.OK("foo"),
				wantErr: false
			},
			{
				name:    "ERR: getFoo",
				getFoo:  tpp.Err(),
				wantErr: true,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				mock := mymocks.NewBar(t)
				tt.getFoo.Expectorise(mock.EXPECT().GetFoo())

				subject := subject.New(mock)
				err := subject.XXX()

				require.Equal(t, tt.wantErr, err != nil)
			})
		}
	}
*/

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	testifymock "github.com/stretchr/testify/mock"
)

var errTest = errors.New("TEST ERROR")

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

// Expect represents an expectation for use in configuration-driven tests.
// This binds together whether a mock call should be expected, what it should
// return, and whether it should return an error.
type Expect struct {
	// Expected determines whether handled mocks should be expected, where:
	//   - true  => The mock must be called
	//   - false => The mock must not be called
	//   - nil   => The mock may or may not be called
	Expected *bool

	// Return are the *non-error* returns for the mock.
	//
	// If you want an error to be returned, set Expect.Err = true.
	Return []any

	// Err determines whether we should append an error to the mock's returns.
	//
	// This is separated out from `Return` for convenience and readability.
	Err bool
}

// OK returns an Expect with the given return and no error.
func OK(returns ...any) Expect {
	return Expect{
		Expected: True(),
		Return:   returns,
		Err:      false,
	}
}

// Err returns an Expect with an error.
func Err() Expect {
	return Expect{
		Expected: True(),
		Err:      true,
	}
}

// Unexpected returns an Expect which is unexpected.
func Unexpected() Expect {
	return Expect{
		Expected: False(),
	}
}

// Injecting returns a new Expect with the given |ret| injected into its Return.
//
// This is useful in cases where the test case itself does not care about the
// returned values, but the meta-test does. For example, where the expected call
// is a factory which returns some also-mocked object constructed within the
// meta-test.
func (e *Expect) Injecting(ret any) *Expect {
	return &Expect{
		Expected: e.Expected,
		Return:   append(e.Return, ret),
		Err:      e.Err,
	}
}

// Mocker represents a Mockery mock.
type Mocker interface {
	Maybe() *testifymock.Call
	Unset() *testifymock.Call

	// We can't specify Return() because different mocks have different returns.
}

// Expectorise configures the given |mock| according to the behaviour specified
// in the Expect.
//
// I.e., whether it should be expected to be called, its return values, and
// whether it should return an error. This is to remove boilerplate code.
//
// If Expect.Expected is true the mock must be called. If false, the mock must
// not be called. If nil, the mock may be called.
//
// If Expect.Return is nil, Expectorise will configure the |mock| to return
// zero values of the returned types. If Expect.Err is set, Expectorise will
// ensure that one of these zero values is set to a non-nil error.
//
// If Expect.Return is non-nil, Expectorise will configure the |mock| to return
// Expect.Return. If Expect.Err is also set, Expectorise will append a non-nil
// error to the returned values.
func (e *Expect) Expectorise(mock Mocker) {
	// Because the type of "Return" depends on the thing being mocked, we have to
	// dynamically get it with reflection...
	returnMethod := reflect.ValueOf(mock).MethodByName("Return")
	if !returnMethod.IsValid() {
		panic("given mock has no return")
	}
	returnMethodType := returnMethod.Type()
	returnMethodNArgs := returnMethodType.NumIn()

	returnMethodIncludesErr := false
	for i := 0; i < returnMethodNArgs; i++ {
		if returnMethodType.In(i).Name() == "error" {
			returnMethodIncludesErr = true
			break
		}
	}

	if e.Expected != nil && !*e.Expected {
		mock.Unset()
		return
	}

	if e.Expected == nil {
		mock.Maybe()
	}

	if e.Return == nil {
		// The Expect hasn't specified the return values, so construct empty ones
		emptyArgs := make([]reflect.Value, returnMethodNArgs)
		for i := 0; i < returnMethodNArgs; i++ {
			argType := returnMethodType.In(i)
			if argType.Name() == "error" {
				if e.Err {
					emptyArgs[i] = reflect.ValueOf(errTest)
				} else {
					emptyArgs[i] = reflect.Zero(argType)
				}
			} else {
				zeroValue := reflect.Zero(argType)
				emptyArgs[i] = zeroValue
			}
		}
		returnMethod.Call(emptyArgs)
		return
	}

	// The Expect has specified return values: use those.
	returns := append([]any{}, e.Return...)
	if e.Err {
		returns = append(returns, errTest)
	} else if returnMethodIncludesErr {
		var err error
		returns = append(returns, err)
	}
	givenArgs, err := toReflectValues(returns, returnMethod)
	if err != nil {
		panic(fmt.Sprintf("toReflectValues failed to transform return values: %s", err))
	}
	returnMethod.Call(givenArgs)
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
		Expected: False(),
	}
}

// OKs returns an Expects with the given calls.
func OKs(calls []Call) Expects {
	return Expects{
		Expected: True(),
		Calls:    calls,
	}
}

// Errs returns an Expects with an error.
func Errs() Expects {
	return Expects{
		Expected: True(),
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
	inputArgs, err := toReflectValues(args, fn)
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

// toReflectValues transforms the |args| of the |method| from `[]any` to
// `[]reflect.Value`.
func toReflectValues(args []any, method reflect.Value) ([]reflect.Value, error) {
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

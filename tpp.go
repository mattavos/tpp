// t++ (t plus plus)

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
				getFoo:  tpp.Return("foo", nil),
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
	"github.com/pkg/errors"

	testifymock "github.com/stretchr/testify/mock"
)

// Return returns an Expect with the given return values.
func Return(returns ...any) Expect {
	return Expect{
		Expected:    ptr(true),
		Return:      returns,
		Err:         nil,
		exactReturn: true,
	}
}

// OK returns an Expect with the given return and no error.
//
// Any error values on mockery mocks will be automatically zero valued once the
// Expect is passed to Expectorise.
func OK(returns ...any) Expect {
	return Expect{
		Expected: ptr(true),
		Return:   returns,
		Err:      nil,
	}
}

// Err returns an Expect with a generic test error.
func Err() Expect {
	return Expect{
		Expected: ptr(true),
		Err:      errDefault,
	}
}

// ErrWith returns an Expect with the given error.
func ErrWith(e error) Expect {
	return Expect{
		Expected: ptr(true),
		Err:      e,
	}
}

// Unexpected returns an Expect which is unexpected.
func Unexpected() Expect {
	return Expect{
		Expected: ptr(false),
	}
}

// Given starts a builder with args which will ultimately configure an Expect.
func Given(args ...any) *callBuilder {
	return &callBuilder{
		args: args,
	}
}

type callBuilder struct {
	args []any
}

// Return returns an Expect with the given returns and args from Given().
func (c *callBuilder) Return(returns ...any) Expect {
	return Expect{
		Expected:        ptr(true),
		argReplacements: c.args,
		Return:          returns,
		exactReturn:     true,
	}
}

// Arg represents an argument placeholder to be used with tpp.Given(). This can
// be used in two ways.
//
// First, if specified as part of a mock call's arguments, it will be filled in
// when the Expect Expectorises the mock. This is used in conjunction with
// tpp.Given(xxx).Return(yyy) to allow for dynamic mocking.
//
// For example, a tpp.Expect that's set up like so:
//
//	expectFoo: tpp.Given("foo", 1).Return(xyz)
//
// Should be Expectorised like this:
//
//	expectFoo.Expectorise(mymock.EXPECT().Foo(tpp.Arg(), tpp.Arg()))
//
// The "foo" and 1 will then be injected in the place of the tpp.Arg()s.
//
// Secondly, if specified as part of an Expect's args it can stand as a
// placeholder argument that will be filled in by the meta-test body.
//
// For example, a tpp.Expect that's set up like so:
//
//	expectFoo: tpp.Given(tpp.Arg(), 1).Return(xyz)
//
// Can be Expectorised like this:
//
//	expectFoo.Expectorise(mymock.EXPECT().Foo(whateverYouWant, tpp.Arg()))
func Arg() templateArg {
	return templateArg{}
}

type templateArg struct{}

// -----------------------------------------------------------------------------
// Expect ----------------------------------------------------------------------
// -----------------------------------------------------------------------------

// Expect represents an expectation for use in configuration-driven tests.
// This binds together whether a mock call should be expected, what it should
// return, and whether it should return an error.
type Expect struct {
	// Expected determines whether handled mocks should be expected, where:
	//   - true  => The mock must be called
	//   - false => The mock must not be called
	//   - nil   => The mock may or may not be called
	Expected *bool

	// Return are the return arguments for the mock.
	//
	// In some cases, these can be incomplete. For example, if the Expect is
	// created by OK(), only non-error returns will be specified and the errors
	// will be zeroed out during Expectorise. Similarly, if the Expect is created
	// by Err(), only the Err field will be set and the non-error returns will be
	// zeroed out.
	Return []any

	// Err determines the error which will be appended to the mock's returns.
	// If it is nil, no error will be appended.
	//
	// This is separated out from `Return` for convenience and readability.
	Err error

	// argReplacements are optional arguments which we will use to replace any
	// tpp.Arg values in the mock.Arguments. These will *only* be added to the mock
	// call if its arguments are specified as tpp.Arg().
	//
	// See tpp.Arg() for more info.
	argReplacements []any

	// nTimes indicates that the mock should only return the indicated number
	// of times.
	nTimes int

	// exactReturn determines that the mock should be configured to return exactly
	// what is specified by Return, and not have errors zero-valued out if
	// unspecified. This is here temporarily to support Return() and
	// Given(xxx).Return(yyy), while also maintaining backwards compatibility,
	// since e.g., OK() implicitly zeroes out errors. This isn't the preferred way
	// of doing things, so we'll move towards being more explicit.
	exactReturn bool
}

// Injecting returns a new Expect with the given |ret| injected into its Return.
//
// This is useful in cases where the test case itself does not care about the
// returned values, but the meta-test does. For example, where the expected call
// is a factory which returns some also-mocked object constructed within the
// meta-test.
func (e *Expect) Injecting(ret any) *Expect {
	return &Expect{
		Expected:    e.Expected,
		Return:      append(e.Return, ret),
		Err:         e.Err,
		exactReturn: e.exactReturn,
	}
}

// Times indicates that the mock should only return the indicated number of times.
// of times.
func (e Expect) Times(n int) Expect {
	e.nTimes = n
	return e
}

// Once indicates that the mock should only return once.
func (e Expect) Once() Expect {
	return e.Times(1)
}

// expectoriseOptions is used to configure Expectorise and ExpectoriseMulti.
type expectoriseOptions struct {
	defaultReturns []any
}

type ExpectoriseOption func(*expectoriseOptions)

// WithDefaultReturns sets default returns to be used when configuring the mock.
//
// These will be provided instead of zero-value returns where returns are not
// otherwise provided by the Expect.
func WithDefaultReturns(returns ...any) ExpectoriseOption {
	return func(opt *expectoriseOptions) {
		opt.defaultReturns = returns
	}
}

// MockCall represents a Mockery mock.
type MockCall interface {
	Maybe() *testifymock.Call
	Unset() *testifymock.Call
	Times(int) *testifymock.Call

	// We can't specify Return() because different mocks have different returns.
	// Instead, we use reflection. See reflect.go.
}

// Expectorise configures the given mock call according to the behaviour
// specified in the Expect.
//
// I.e., whether it should be expected to be called, its args (if specified,
// its return values, and whether it should return an error.
//
// Iff any of the arguments to the mock are tpp.Arg(), they will be replaced by
// the arguments in Expect.Args.
//
// If Expect.Expected is true the mock must be called. If false, the mock must
// not be called. If nil, the mock may be called.
//
// If Expect.Return is nil, Expectorise will configure the mock to return
// zero values of the returned types. If Expect.Err is set, Expectorise will
// ensure that one of these zero values is set to a non-nil error.
//
// If Expect.Return is non-nil, Expectorise will configure the mock to return
// Expect.Return. If Expect.Err is also set, Expectorise will append a non-nil
// error to the returned values.
func (e *Expect) Expectorise(mock MockCall, options ...ExpectoriseOption) {
	// Parse options
	var opts expectoriseOptions
	for _, o := range options {
		o(&opts)
	}

	if e.Expected != nil && !*e.Expected {
		unsetMock(mock)
		return
	}

	if e.Expected == nil {
		mock.Maybe()
	}

	if e.nTimes > 0 {
		mock.Times(e.nTimes)
	}

	rmock, err := newReflectedMockCall(mock)
	if err != nil {
		panic(err)
	}

	// Replace any args that have been specified with tpp.Arg() with the args
	// specified on the Expect.
	args, err := rmock.GetArguments()
	if err != nil {
		panic(err)
	}
	var newargs []any
	for i, arg := range args {
		if _, ok := arg.(templateArg); ok {
			if i >= len(e.argReplacements) {
				// We've ran out of supplied args. This happens commonly, since the
				// Expect might be empty or an error, but the test-body specifies
				// a tpp.Arg() for the mock arguments. Fall back to mock.Anything.
				newargs = append(newargs, testifymock.Anything)
			} else {
				newargs = append(newargs, e.argReplacements[i])
				i++
			}
		} else {
			newargs = append(newargs, arg)
		}
	}
	rmock.SetArguments(newargs)

	switch {
	case e.Return != nil:
		err := rmock.CallReturn(e.Return, e.Err, !e.exactReturn)
		if err != nil {
			panic(err)
		}

	case e.Err != nil:
		rmock.CallReturnEmpty(e.Err)

	case opts.defaultReturns != nil:
		err := rmock.CallReturn(opts.defaultReturns, nil, false)
		if err != nil {
			panic(err)
		}

	default:
		rmock.CallReturnEmpty(nil)
	}
}

// -----------------------------------------------------------------------------
// ExpectoriseMulti ------------------------------------------------------------
// -----------------------------------------------------------------------------

// ExpectoriseMulti configures the given mock calls according to the behaviour
// specified in the []Expect.
//
// The callFn will be called once for each Expect in the given []Expect.
// This enables multiple mock calls to be configured.
//
// For example:
//
//	ee := []tpp.Expect{
//		tpp.Return(123),
//		tpp.Return(456),
//		tpp.Return(789),
//	}
//	tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
//		return m.EXPECT().MyFn(1)
//	})
//
// A nil slice will result in a zero-valued return, just like a zero-valued
// Expect.
//
// An empty slice will result in the mock call being unexpected.
//
// For more info, see Expect.Expectorise.
func ExpectoriseMulti(ee []Expect, callFn func() MockCall, options ...ExpectoriseOption) {
	// Parse options
	var opts expectoriseOptions
	for _, o := range options {
		o(&opts)
	}

	// If there are no Expects in the slice, then set up a mock which accepts
	// anything and will return an empty/default return.
	if ee == nil {
		call := callFn()
		call.Maybe()

		rmock, err := newReflectedMockCall(call)
		if err != nil {
			panic(err)
		}

		// Replace tpp.Arg()s with mock.Anything.
		args, err := rmock.GetArguments()
		if err != nil {
			panic(err)
		}
		var newargs []any
		for _, arg := range args {
			if _, ok := arg.(templateArg); ok {
				newargs = append(newargs, testifymock.Anything)
			} else {
				newargs = append(newargs, arg)
			}
		}
		rmock.SetArguments(newargs)

		// Return either the specified default, or empty.
		if opts.defaultReturns != nil {
			err := rmock.CallReturn(opts.defaultReturns, nil, false)
			if err != nil {
				panic(err)
			}
		} else {
			rmock.CallReturnEmpty(nil)
		}
		return
	}

	for _, e := range ee {
		e := e
		call := callFn()
		e.Expectorise(call)
	}
}

// -----------------------------------------------------------------------------
// Unexported Helpers ----------------------------------------------------------
// -----------------------------------------------------------------------------

// unsetMock unsets a mock. This is necessary because testify's mock.Call.Unset()
// does not gracefully handle the case where we have an argument matcher.
func unsetMock(mock MockCall) {
	// mock may be a type that wraps a testify mock.Call, we can use Maybe to extract it, and then unset.
	if call := mock.Maybe(); call != nil {
		safeUnsetCall(call)
	} else {
		mock.Unset()
	}
}

// safeUnsetCall safely unsets a mock call from its parent mock object.
func safeUnsetCall(call *testifymock.Call) {
	parent := call.Parent
	if parent == nil {
		return
	}

	calls := parent.ExpectedCalls
	if calls == nil {
		return
	}

	for i, c := range calls {
		if c == call {
			newCalls := append(calls[:i], calls[i+1:]...)
			parent.ExpectedCalls = newCalls
			break
		}
	}
}

var errDefault = errors.New("ERROR")

func ptr[T any](t T) *T {
	return &t
}

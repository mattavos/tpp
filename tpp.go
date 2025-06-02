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
	"github.com/pkg/errors"

	testifymock "github.com/stretchr/testify/mock"
)

// Return returns an Expect with the given return values.
func Return(returns ...any) Expect {
	return Expect{
		Expected: ptr(true),
		Return:   returns,
		Err:      nil,
	}
}

// OK returns an Expect with the given return and no error.
//
// Any error values on mockery mocks will be automatically zero valued.
func OK(returns ...any) Expect {
	return Return(returns...)
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
		ArgReplacements: c.args,
		Return:          returns,
	}
}

// Arg represents an argument placeholder that will be filled in when the Expect
// is Expectorised. This is used in conjunction with tpp.Given(xxx).Return(yyy)
// to allow for dynamic mocking.
//
// For example, a tpp.Expect that's set up like so:
//
//	expectFoo := tpp.With("foo", 1).Return(xyz)
//
// Should be Expectorised like this:
//
//	expectFoo.Expectorise(mymock.EXPECT().Foo(tpp.Arg(), tpp.Arg()))
//
// The "foo" and 1 will then be injected in the place of the tpp.Arg()s.
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

	// ArgReplacements are optional arguments which we will use to replace any
	// tpp.Arg values in the mock.Arguments. These will *only* be added to the mock
	// call if its arguments are specified as tpp.Arg().
	//
	// See tpp.Arg() for more info.
	ArgReplacements []any

	// Return are the *non-error* returns for the mock.
	//
	// If you want an error to be returned, place it in Expect.Err.
	Return []any

	// Err determines the error which will be appended to the mock's returns.
	// If it is nil, no error will be appended.
	//
	// This is separated out from `Return` for convenience and readability.
	Err error

	// NTimes indicates that the mock should only return the indicated number
	// of times.
	NTimes int
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

// Times indicates that the mock should only return the indicated number of times.
// of times.
func (e Expect) Times(n int) Expect {
	e.NTimes = n
	return e
}

// Once indicates that the mock should only return once.
func (e Expect) Once() Expect {
	return e.Times(1)
}

// expectoriseOption is used to configure Expectorise and ExpectoriseMulti.
type expectoriseOption struct {
	defaultReturns []any
}

// WithDefaultReturns sets default returns to be used when configuring the mock.
//
// These will be provided instead of zero-value returns where returns are not
// otherwise provided by the Expect.
func WithDefaultReturns(returns ...any) func(*expectoriseOption) {
	return func(opt *expectoriseOption) {
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
func (e *Expect) Expectorise(mock MockCall, options ...func(*expectoriseOption)) {
	// Parse options
	var opts expectoriseOption
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

	if e.NTimes > 0 {
		mock.Times(e.NTimes)
	}

	// TODO: because of lack of type safety, people are going to both:
	//   (a) pass in the wrong number of arguments/returns, and
	//   (b) pass in the wrong type of arguments/returns
	// especially due to refactoring code. We need to make sure that the
	// errors one gets back in these two cases are exeptionally helpful.

	rmock, err := newReflectedMockCall(mock)
	if err != nil {
		panic(err)
	}

	if e.Expected != nil && *e.Expected {
		// Replace any args that have been specified with tpp.Arg() with the args
		// specified on the Expect.
		args, err := rmock.GetArguments()
		if err != nil {
			panic(err)
		}

		// TODO: panic with a helpful message if e.Args is set but none
		// of the rmock arguments are templateArgs. This is a mistake!

		var newargs []any
		var idx int
		for _, arg := range args {
			if _, ok := arg.(templateArg); ok {
				if idx >= len(e.ArgReplacements) {
					// We've ran out of args: this happens if we specified an error in the
					// Expect and the test still put a placeholder in. All good.
					newargs = append(newargs, testifymock.Anything)
				} else {
					newargs = append(newargs, e.ArgReplacements[idx])
					idx++
				}
			} else {
				newargs = append(newargs, arg)
			}
		}
		rmock.SetArguments(newargs)
	}

	switch {
	case e.Return != nil:
		err := rmock.CallReturn(e.Return, e.Err)
		if err != nil {
			panic(err)
		}

	case e.Err != nil:
		rmock.CallReturnEmpty(e.Err)

	case opts.defaultReturns != nil:
		err := rmock.CallReturn(opts.defaultReturns, nil)
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
func ExpectoriseMulti(ee []Expect, callFn func() MockCall, options ...func(*expectoriseOption)) {
	// Parse options
	var opts expectoriseOption
	for _, o := range options {
		o(&opts)
	}

	// If there are no Expects in the slice, then set up an empty/default return.
	if ee == nil {
		call := callFn()
		call.Maybe()

		rmock, err := newReflectedMockCall(call)
		if err != nil {
			panic(err)
		}
		if opts.defaultReturns != nil {
			err := rmock.CallReturn(opts.defaultReturns, nil)
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

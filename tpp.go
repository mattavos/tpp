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
// TODO: think about how this fits in now that we have to handle
// Given(xxx).Return(yyy). Is it just equivalent?
func OK(returns ...any) Expect {
	return Return(returns...)
}

// Err returns an Expect with a generic test error.
func Err() Expect {
	return Expect{
		Expected: ptr(true),
		Err:      errTest,
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

func Given(args ...any) *callBuilder {
	return &callBuilder{
		args: args,
	}
}

type callBuilder struct {
	args []any
}

func (c *callBuilder) Return(returns ...any) Expect {
	return Expect{
		Expected: ptr(true),
		Args:     c.args,
		Return:   returns,
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

// Expect represents an expectation for use in configuration-driven tests.
// This binds together whether a mock call should be expected, what it should
// return, and whether it should return an error.
type Expect struct {
	// Expected determines whether handled mocks should be expected, where:
	//   - true  => The mock must be called
	//   - false => The mock must not be called
	//   - nil   => The mock may or may not be called
	Expected *bool

	// Args are the arguments the handled mock should expect. These will only be
	// added to the mock call if its arguments are specified as tpp.Arg().
	// See tpp.Arg() for more info.
	Args []any

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

// TODO: Dave Cheney style Option for things like default returns
//
//func WithDefaultReturns() func(*option) {
//}

// Mocker represents a Mockery mock.
type Mocker interface {
	Maybe() *testifymock.Call
	Unset() *testifymock.Call
	Times(int) *testifymock.Call

	// We can't specify Return() because different mocks have different returns.
	// Instead, we use reflection. See reflect.go.
}

// Expectorise configures the given mock according to the behaviour specified
// in the Expect.
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
func (e *Expect) Expectorise(mock Mocker) {
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
				if idx >= len(e.Args) {
					// We've ran out of args: this happens if we specified an error in the
					// Expect and the test still put a placeholder in. All good.
					newargs = append(newargs, testifymock.Anything)
				} else {
					newargs = append(newargs, e.Args[idx])
					idx++
				}
			} else {
				newargs = append(newargs, arg)
			}
		}
		rmock.SetArguments(newargs)
	}

	if e.Return == nil {
		rmock.CallReturnEmpty(e.Err)
	} else {
		err := rmock.CallReturn(e.Return, e.Err)
		if err != nil {
			panic(err)
		}
	}
}

type ExpectMany []Expect

func (em ExpectMany) Expectorise(mock Mocker) {
	if em == nil {
		mock.Maybe()
		rmock, err := newReflectedMockCall(mock)
		if err != nil {
			panic(err)
		}
		rmock.CallReturnEmpty(nil)
		return
	}
	for _, e := range em {
		e.Expectorise(mock)
	}
}

// unsetMock unsets a mock. This is necessary because testify's mock.Call.Unset()
// does not gracefully handle the case where we have an argument matcher.
func unsetMock(mock Mocker) {
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

var errTest = errors.New("TEST ERROR")

func ptr[T any](t T) *T {
	return &t
}

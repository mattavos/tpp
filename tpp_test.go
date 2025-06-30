package tpp_test

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"unsafe"

	"github.com/pkg/errors"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattavos/tpp"
	"github.com/mattavos/tpp/testdata"
)

type exampleCall struct {
	name    string
	args    []any
	returns []any
}

func TestExpect(t *testing.T) {
	// We use this dummy testing.T to pass into the code under test. We're testing
	// test code, so we don't want the failed expectations within the code we're
	// testing to fail *our* tests!
	_t := func() *testing.T {
		return &testing.T{}
	}

	for _, tt := range []struct {
		name            string
		expectoriseCall func(e tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock)
		argTypes        []string
		defaultArgs     []any
		returnTypes     []string
		defaultReturns  []any
		examples        []exampleCall
	}{
		{
			name:           "func(a, b int) (int, error)",
			argTypes:       []string{"int", "int"},
			defaultArgs:    []any{1, 2},
			returnTypes:    []string{"int", "error"},
			defaultReturns: []any{1, error(nil)},
			examples: []exampleCall{
				{
					name:    "ok",
					args:    []any{1, 2},
					returns: []any{123, error(nil)},
				},
				{
					name:    "err",
					args:    []any{1, 2},
					returns: []any{0, errTest},
				},
			},
			expectoriseCall: func(
				expect tpp.Expect,
				args []any,
				opts ...tpp.ExpectoriseOption,
			) (*testifymock.Call, *testifymock.Mock) {
				must(len(args) == 2, "expectoriseCall: wrong arg count")
				mock := testdata.NewMockIntyThing(_t())
				call := mock.EXPECT().DoThing(args[0], args[0])
				expect.Expectorise(call, opts...)
				return call.Call, &mock.Mock
			},
		},
		{
			name:           "func(context.Context, *Struct) (*Struct, error)",
			argTypes:       []string{"Context", "*Struct"},
			defaultArgs:    []any{context.Background(), &testdata.Struct{A: 1, B: 2}},
			returnTypes:    []string{"*Struct", "error"},
			defaultReturns: []any{&testdata.Struct{A: 1, B: 2}, error(nil)},
			examples: []exampleCall{
				{
					name:    "ok",
					args:    []any{context.Background(), &testdata.Struct{A: 1, B: 2}},
					returns: []any{&testdata.Struct{A: 1, B: 2}, error(nil)},
				},
				{
					name:    "err",
					args:    []any{context.Background(), &testdata.Struct{A: 3, B: 4}},
					returns: []any{(*testdata.Struct)(nil), errTest},
				},
			},
			expectoriseCall: func(
				expect tpp.Expect,
				args []any,
				opts ...tpp.ExpectoriseOption,
			) (*testifymock.Call, *testifymock.Mock) {
				must(len(args) == 2, "expectoriseCall: wrong arg count")
				mock := testdata.NewMockStructyThing(_t())
				call := mock.EXPECT().DoThing(args[0], args[1])
				expect.Expectorise(call, opts...)
				return call.Call, &mock.Mock
			},
		},
		{
			name: "Testify mock",
			// These tests check the behaviour of Expectorise when it's passed a "bare"
			// not-wrapped mock.Call, such as you get from mock.On("Foo", xxx, yyy).
			//
			// This isn't the most common case, since we generally pass in the wrapped
			// call you get from mock.EXPECT().Foo(xxx, yyy). We only really allow this
			// because there's no good way to reject it (maybe we could do a runtime
			// check? But that's a bit gross...).
			argTypes:       []string{},
			defaultArgs:    []any{},
			returnTypes:    []string{},
			defaultReturns: []any{},
			examples: []exampleCall{
				{
					name:    "func()",
					args:    []any{},
					returns: []any{},
				},
				{
					name:    "func(int) (int)",
					args:    []any{1},
					returns: []any{1},
				},
				{
					name:    "func(int) (int, error)",
					args:    []any{1},
					returns: []any{1, error(nil)},
				},
				{
					name:    "func(int) ([]int, error)",
					args:    []any{1},
					returns: []any{[]int{1, 2, 3}, error(nil)},
				},
				{
					name:    "func(Context, *Struct) (*Struct, error)",
					args:    []any{context.Background(), &testdata.Struct{A: 1, B: 2}},
					returns: []any{&testdata.Struct{A: 1, B: 2}, error(nil)},
				},
			},
			expectoriseCall: func(
				expect tpp.Expect,
				args []any,
				opts ...tpp.ExpectoriseOption,
			) (*testifymock.Call, *testifymock.Mock) {
				mock := (&testifymock.Mock{})
				call := mock.On("Test", args...)
				expect.Expectorise(call, opts...)
				return call, mock
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			t.Run("Zero value gets empty return", func(t *testing.T) {
				var expect tpp.Expect
				call, _ := tt.expectoriseCall(expect, tt.defaultArgs)

				require.Len(t, call.ReturnArguments, len(tt.returnTypes))
				for _, ret := range call.ReturnArguments {
					require.Empty(t, ret)
				}
			})

			t.Run("Zero value is Maybe()d", func(t *testing.T) {
				var expect tpp.Expect
				call, _ := tt.expectoriseCall(expect, tt.defaultArgs)
				require.True(t, isCallOptional(call))
			})

			for _, example := range tt.examples {
				t.Run("Return() sets up return: "+example.name, func(t *testing.T) {
					call, _ := tt.expectoriseCall(tpp.Return(example.returns...), tt.defaultArgs)
					require.Equal(t, toArgs(example.returns...), call.ReturnArguments)
				})

				t.Run("Return() is not Maybe()d: "+example.name, func(t *testing.T) {
					call, _ := tt.expectoriseCall(tpp.Return(example.returns...), tt.defaultArgs)
					require.False(t, isCallOptional(call))
				})

				{
					// tpp.OK() will automagically nil out error values in the return, so
					// we need a version of the return example without errors specified.
					returnsWithoutErrTypes, returnsWithNilledErrs := okReturns(example.returns)

					t.Run("OK() sets up return: "+example.name, func(t *testing.T) {
						call, _ := tt.expectoriseCall(
							tpp.OK(returnsWithoutErrTypes...),
							tt.defaultArgs,
						)
						require.Equal(t, toArgs(returnsWithNilledErrs...), call.ReturnArguments)
					})

					t.Run("OK() is not Maybe()d: "+example.name, func(t *testing.T) {
						call, _ := tt.expectoriseCall(
							tpp.OK(returnsWithoutErrTypes...),
							tt.defaultArgs,
						)
						require.False(t, isCallOptional(call))
					})
				}

				if slices.Contains(tt.returnTypes, "error") {
					t.Run("Err() sets up err return", func(t *testing.T) {
						call, _ := tt.expectoriseCall(tpp.Err(), tt.defaultArgs)

						require.Len(t, call.ReturnArguments, len(tt.returnTypes))
						for i, ret := range tt.returnTypes {
							if ret == "error" {
								_, ok := call.ReturnArguments[i].(error)
								require.True(t, ok)
							} else {
								require.Empty(t, call.ReturnArguments[i])
							}
						}
					})

					t.Run("Err() is not Maybe()d", func(t *testing.T) {
						call, _ := tt.expectoriseCall(tpp.Err(), tt.defaultArgs)
						require.False(t, isCallOptional(call))
					})

					t.Run("ErrWith() sets up err return", func(t *testing.T) {
						call, _ := tt.expectoriseCall(tpp.ErrWith(errTest), tt.defaultArgs)

						for i, ret := range tt.returnTypes {
							if ret == "error" {
								retErr, ok := call.ReturnArguments[i].(error)
								require.True(t, ok)
								require.Equal(t, errTest, retErr)
							} else {
								require.Empty(t, call.ReturnArguments[i])
							}
						}
					})

					t.Run("ErrWith() is not Maybe()d", func(t *testing.T) {
						call, _ := tt.expectoriseCall(tpp.ErrWith(errTest), tt.defaultArgs)
						require.False(t, isCallOptional(call))
					})
				}

				for _, example := range tt.examples {
					t.Run("Given().Return() sets up args and return", func(t *testing.T) {
						expect := tpp.Given(example.args...).Return(example.returns...)

						// When the arguments are specified by Given in the Expect, they're
						// specified as tpp.Arg() placeholders when we set up the mock.
						// Then Expectorise will replace them with the args from Expect.
						callArgs := placeholders(len(example.args))

						c, _ := tt.expectoriseCall(expect, callArgs)

						require.Equal(t, toArgs(example.args...), c.Arguments)
						require.Equal(t, toArgs(example.returns...), c.ReturnArguments)
					})

					t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
						expect := tpp.Given(example.args...).Return(example.returns...)
						call, _ := tt.expectoriseCall(expect, placeholders(len(example.args)))
						require.False(t, isCallOptional(call))
					})

					t.Run("Given().Return() can take mock.Anything as an arg", func(t *testing.T) {
						args := replaceLast(example.args, testifymock.Anything)

						expect := tpp.Given(args...).Return(example.returns...)
						call, _ := tt.expectoriseCall(expect, placeholders(len(example.args)))

						require.Equal(t, toArgs(args...), call.Arguments)
					})

					t.Run("Given().Return() can handle tpp.Arg() injection", func(t *testing.T) {
						// tpp.Arg() can be used to signify that the value will be filled in
						// later by the meta-test. So here we Expect using tpp.Arg() as the
						// final arg, and then when we call the mock we fill that value in,
						// and pass the other args as tpp.Arg() (since they *were* specified
						// by the Expect).
						expectArgs := replaceLast(example.args, tpp.Arg())
						callArgs := replaceAllButLast(example.args, tpp.Arg())

						expect := tpp.Given(expectArgs...).Return(example.returns...)
						call, _ := tt.expectoriseCall(expect, callArgs)

						require.Equal(t, toArgs(example.args...), call.Arguments)
						require.Equal(t, toArgs(example.returns...), call.ReturnArguments)
					})
				}
			}

			t.Run("Unexpected() unsets mock", func(t *testing.T) {
				expect := tpp.Unexpected()
				_, mock := tt.expectoriseCall(expect, placeholders(len(tt.defaultArgs)))
				require.Empty(t, mock.ExpectedCalls)
			})

			t.Run("Once() sets repeatability", func(t *testing.T) {
				expect := tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Once()
				call, _ := tt.expectoriseCall(expect, placeholders(len(tt.defaultArgs)))
				require.Equal(t, 1, call.Repeatability)
			})

			t.Run("Times() sets repeatability", func(t *testing.T) {
				expect := tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Times(42)
				call, _ := tt.expectoriseCall(expect, placeholders(len(tt.defaultArgs)))
				require.Equal(t, 42, call.Repeatability)
			})

			t.Run("Injecting() adds to return", func(t *testing.T) {
				expect := tpp.OK( /* provided by injection */ )
				for _, ret := range tt.defaultReturns {
					expect = *expect.Injecting(ret)
				}
				call, _ := tt.expectoriseCall(expect, tt.defaultArgs)
				require.Equal(t, toArgs(tt.defaultReturns...), call.ReturnArguments)
			})

			t.Run("Injecting() works with zero value Expect", func(t *testing.T) {
				var expect tpp.Expect
				for _, ret := range tt.defaultReturns {
					expect = *expect.Injecting(ret)
				}
				call, _ := tt.expectoriseCall(expect, tt.defaultArgs)
				require.Equal(t, toArgs(tt.defaultReturns...), call.ReturnArguments)
			})

			t.Run("Arg() gets replaced with mock.Anything for empty expects", func(t *testing.T) {
				var expect tpp.Expect
				call, _ := tt.expectoriseCall(expect, placeholders(len(tt.defaultArgs)))
				require.Equal(t, toArgs(anythings(len(tt.defaultArgs))...), call.Arguments)
			})

			t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
				var expect tpp.Expect
				call, _ := tt.expectoriseCall(
					expect,
					tt.defaultArgs,
					tpp.WithDefaultReturns(tt.defaultReturns...),
				)
				require.Equal(t, toArgs(tt.defaultReturns...), call.ReturnArguments)
			})

			t.Run(
				"WithDefaultReturns() adds to return only if Expect empty",
				func(t *testing.T) {
					type wrong struct{}

					expect := tpp.Return(tt.defaultReturns...)
					call, _ := tt.expectoriseCall(
						expect,
						tt.defaultArgs,
						tpp.WithDefaultReturns(wrong{}),
					)
					require.Equal(t, toArgs(tt.defaultReturns...), call.ReturnArguments)
				},
			)

			if len(tt.defaultReturns) > 0 {
				t.Run(
					"WithDefaultReturns() causes panic if wrong number of args",
					func(t *testing.T) {
						var wrongArgs []any
						for _, a := range tt.defaultReturns {
							// Double sized
							wrongArgs = append(wrongArgs, a)
							wrongArgs = append(wrongArgs, a)
						}

						var expect tpp.Expect
						require.Panics(t, func() {
							tt.expectoriseCall(
								expect,
								tt.defaultArgs,
								tpp.WithDefaultReturns(wrongArgs...),
							)
						})
					},
				)

				t.Run("WithDefaultReturns() causes panic if wrong type args", func(t *testing.T) {
					type wrong struct{}

					var expect tpp.Expect

					require.Panics(t, func() {
						tt.expectoriseCall(
							expect,
							tt.defaultArgs,
							tpp.WithDefaultReturns(wrong{}),
						)
					})
				})
			}

		})
	}
}

// Here are a few cases for a bare testify mock call which aren't caught in the
// generic test above.
func TestExpectWithTestifyMock(t *testing.T) {
	t.Run("Err() sets up err return", func(t *testing.T) {
		c := (&testifymock.Mock{}).On("Test", 1)

		e := tpp.Err()
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 1)
		_, ok := c.ReturnArguments[0].(error)
		require.True(t, ok)
	})

	t.Run("Err() is not Maybe()d", func(t *testing.T) {
		c := (&testifymock.Mock{}).On("Test", 1)

		e := tpp.Err()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("ErrWith() sets up err return", func(t *testing.T) {
		c := (&testifymock.Mock{}).On("Test", 1)

		withErr := errors.New("Everything exploded")
		e := tpp.ErrWith(withErr)
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 1)
		err, ok := c.ReturnArguments[0].(error)
		require.True(t, ok)
		require.Equal(t, withErr, err)
	})

	t.Run("ErrWith() is not Maybe()d", func(t *testing.T) {
		c := (&testifymock.Mock{}).On("Test", 1)

		withErr := errors.New("Everything exploded")
		e := tpp.ErrWith(withErr)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})
}

// We're testing using a mockery mock of an interface which looks like this:
//
//	type IntyThing interface {
//		DoThing(a, b int) (int, error)
//	}
func TestExpectMultiWithMockeryMockIntyThing(t *testing.T) {
	_t := &testing.T{} // dummy testing.T for passing into code under test

	t.Run("Zero value gets empty return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 1)
		for _, a := range m.ExpectedCalls[0].ReturnArguments {
			require.Empty(t, a)
		}
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 1)
		require.True(t, isCallOptional(m.ExpectedCalls[0]))
	})

	t.Run("Empty unsets mock", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			/* No expectations */
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Return() sets up calls", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Return(123, nil),
			tpp.Return(456, nil),
			tpp.Return(789, nil),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(i, i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0, 0), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(123, nil), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(1, 1), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(456, nil), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(2, 2), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(789, nil), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Return() calls aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Return(123, nil),
			tpp.Return(456, nil),
			tpp.Return(789, nil),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(0, 0)
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("OK() sets up calls", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.OK(123),
			tpp.OK(456),
			tpp.OK(789),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(i, i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0, 0), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(123, nil), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(1, 1), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(456, nil), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(2, 2), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(789, nil), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("OK() calls aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.OK(123),
			tpp.OK(456),
			tpp.OK(789),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(i, i)
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Err() sets up err returns", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Err(),
			tpp.Err(),
			tpp.Err(),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(i, i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0, 0), m.ExpectedCalls[0].Arguments)
		is.Empty(m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		is.True(ok)

		is.Equal(toArgs(1, 1), m.ExpectedCalls[1].Arguments)
		is.Empty(m.ExpectedCalls[1].ReturnArguments[0])
		_, ok = m.ExpectedCalls[1].ReturnArguments[1].(error)
		is.True(ok)

		is.Equal(toArgs(2, 2), m.ExpectedCalls[2].Arguments)
		is.Empty(m.ExpectedCalls[1].ReturnArguments[0])
		_, ok = m.ExpectedCalls[2].ReturnArguments[1].(error)
		is.True(ok)
	})

	t.Run("Err() returns aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Err(),
			tpp.Err(),
			tpp.Err(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(0, 0)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("ErrWith() sets up err returns", func(t *testing.T) {
		var (
			errOne   = errors.New("one")
			errTwo   = errors.New("two")
			errThree = errors.New("three")
		)
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.ErrWith(errOne),
			tpp.ErrWith(errTwo),
			tpp.ErrWith(errThree),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(i, i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0, 0), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(0, errOne), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(1, 1), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(0, errTwo), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(2, 2), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(0, errThree), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("ErrWith() returns aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.ErrWith(errors.New("one")),
			tpp.ErrWith(errors.New("two")),
			tpp.ErrWith(errors.New("three")),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(i, i)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Given().Return() sets up args and return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(1, 1).Return(1, nil),
			tpp.Given(2, 2).Return(2, nil),
			tpp.Given(3, 3).Return(3, nil),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(1, 1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(1, nil), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(2, 2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(2, nil), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(3, 3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(3, nil), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() sets up args and return: errors", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(1, 1).Return(1, errTest),
			tpp.Given(2, 2).Return(2, errTest),
			tpp.Given(3, 3).Return(3, errTest),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(1, 1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(1, errTest), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(2, 2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(2, errTest), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(3, 3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(3, errTest), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() can handle tpp.Arg() injection", func(t *testing.T) {
		// tpp.Arg() can be used to signify that the value will be filled in later
		// by the meta-test.
		ee := []tpp.Expect{
			tpp.Given(tpp.Arg(), 1).Return(1, errTest),
			tpp.Given(tpp.Arg(), 2).Return(2, errTest),
			tpp.Given(tpp.Arg(), 3).Return(3, errTest),
		}

		m := testdata.NewMockIntyThing(_t)

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(42, tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(42, 1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(1, errTest), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(42, 2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(2, errTest), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(42, 3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(3, errTest), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(1, 1).Return(1, nil),
			tpp.Given(2, 2).Return(2, nil),
			tpp.Given(3, 3).Return(3, nil),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Unexpected() unsets mock", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Unexpected(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(1, 1).Return(1, nil).Once(),
			tpp.Given(2, 2).Return(2, nil).Once(),
			tpp.Given(3, 3).Return(3, nil).Once(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.Equal(t, 1, c.Repeatability)
		}
	})

	t.Run("Times() sets repeatability", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(1, 1).Return(1, nil).Times(1),
			tpp.Given(2, 2).Return(2, nil).Times(2),
			tpp.Given(3, 3).Return(3, nil).Times(3),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for i, c := range m.ExpectedCalls {
			require.Equal(t, i+1, c.Repeatability)
		}
	})

	t.Run("Arg() gets replaced with mock.Anything for empty expects", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(1, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Len(t, m.ExpectedCalls[0].Arguments, 2)
		for _, arg := range m.ExpectedCalls[0].Arguments {
			require.Equal(t, testifymock.Anything, arg)
		}
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(1, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(
			t,
			testifymock.Arguments([]any{1, errTest}),
			m.ExpectedCalls[0].ReturnArguments,
		)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		ee := []tpp.Expect{
			tpp.Return(123, errTest),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(456, error(nil)))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(
			t,
			testifymock.Arguments([]any{123, errTest}),
			m.ExpectedCalls[0].ReturnArguments,
		)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		ee := []tpp.Expect{
			tpp.Err(),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(456, error(nil)))

		require.Empty(t, m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		require.True(t, ok)
	})

	t.Run("WithDefaultReturns() causes panic if wrong number of args", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		var ee []tpp.Expect

		require.Panics(t, func() {
			tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
				return m.EXPECT().DoThing(1, 2)
			}, tpp.WithDefaultReturns(1, 2, 3, 4, 5))
		})
	})
}

// We're testing using a mockery mock of an interface which looks like this:
//
//	type StructyThing interface {
//		DoThing(context.Context, *Struct) (*Struct, error)
//	}
func TestExpectMultiWithMockeryMockStructyThing(t *testing.T) {
	var (
		_t   = &testing.T{} // dummy testing.T for passing into code under test
		ctx1 = context.WithValue(context.Background(), "key", 1)
		ctx2 = context.WithValue(context.Background(), "key", 2)
		ctx3 = context.WithValue(context.Background(), "key", 3)
		a1   = &testdata.Struct{A: 1, B: 1}
		a2   = &testdata.Struct{A: 2, B: 2}
		a3   = &testdata.Struct{A: 3, B: 3}
		args = []*testdata.Struct{a1, a2, a3}
		r1   = &testdata.Struct{A: 1, B: 2}
		r2   = &testdata.Struct{A: 10, B: 20}
		r3   = &testdata.Struct{A: 100, B: 200}
		err1 = errors.New("1")
		err2 = errors.New("2")
		err3 = errors.New("3")
	)

	t.Run("Zero value gets empty return", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 1)
		for _, a := range m.ExpectedCalls[0].ReturnArguments {
			require.Empty(t, a)
		}
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 1)
		require.True(t, isCallOptional(m.ExpectedCalls[0]))
	})

	t.Run("Empty unsets mock", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			/* No expectations */
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Return() sets up calls", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Return(r1, nil),
			tpp.Return(r2, nil),
			tpp.Return(r3, nil),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(ctx1, args[i])
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx1, args[0]), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(r1, nil), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(ctx1, args[1]), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(r2, nil), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(ctx1, args[2]), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(r3, nil), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Return() calls aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Return(r1, nil),
			tpp.Return(r2, nil),
			tpp.Return(r3, nil),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(context.TODO(), r1)
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("OK() sets up calls", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.OK(r1),
			tpp.OK(r2),
			tpp.OK(r3),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(ctx1, args[i])
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx1, args[0]), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(r1, nil), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(ctx1, args[1]), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(r2, nil), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(ctx1, args[2]), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(r3, nil), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("OK() calls aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.OK(r1),
			tpp.OK(r2),
			tpp.OK(r3),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(context.TODO(), r1)
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Err() sets up err returns", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Err(),
			tpp.Err(),
			tpp.Err(),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(ctx1, args[i])
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx1, args[0]), m.ExpectedCalls[0].Arguments)
		is.Empty(m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		is.True(ok)

		is.Equal(toArgs(ctx1, args[1]), m.ExpectedCalls[1].Arguments)
		is.Empty(m.ExpectedCalls[1].ReturnArguments[0])
		_, ok = m.ExpectedCalls[1].ReturnArguments[1].(error)
		is.True(ok)

		is.Equal(toArgs(ctx1, args[2]), m.ExpectedCalls[2].Arguments)
		is.Empty(m.ExpectedCalls[1].ReturnArguments[0])
		_, ok = m.ExpectedCalls[2].ReturnArguments[1].(error)
		is.True(ok)
	})

	t.Run("Err() returns aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Err(),
			tpp.Err(),
			tpp.Err(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(context.TODO(), r1)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("ErrWith() sets up err returns", func(t *testing.T) {
		var (
			errOne   = errors.New("one")
			errTwo   = errors.New("two")
			errThree = errors.New("three")
		)
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.ErrWith(errOne),
			tpp.ErrWith(errTwo),
			tpp.ErrWith(errThree),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(ctx1, args[i])
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx1, args[0]), m.ExpectedCalls[0].Arguments)
		is.Empty(m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		is.True(ok)

		is.Equal(toArgs(ctx1, args[1]), m.ExpectedCalls[1].Arguments)
		is.Empty(m.ExpectedCalls[1].ReturnArguments[0])
		_, ok = m.ExpectedCalls[1].ReturnArguments[1].(error)
		is.True(ok)

		is.Equal(toArgs(ctx1, args[2]), m.ExpectedCalls[2].Arguments)
		is.Empty(m.ExpectedCalls[1].ReturnArguments[0])
		_, ok = m.ExpectedCalls[2].ReturnArguments[1].(error)
		is.True(ok)
	})

	t.Run("ErrWith() returns aren't Maybe()d", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.ErrWith(errors.New("one")),
			tpp.ErrWith(errors.New("two")),
			tpp.ErrWith(errors.New("three")),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.EXPECT().DoThing(context.TODO(), args[i])
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Given().Return() sets up args and return", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(ctx1, a1).Return(r1, nil),
			tpp.Given(ctx2, a2).Return(r2, nil),
			tpp.Given(ctx3, a3).Return(r3, nil),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx1, a1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(r1, nil), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(ctx2, a2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(r2, nil), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(ctx3, a3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(r3, nil), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() sets up args and return: errors", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(ctx1, a1).Return(nil, err1),
			tpp.Given(ctx2, a2).Return(nil, err2),
			tpp.Given(ctx3, a3).Return(nil, err3),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx1, a1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs((*testdata.Struct)(nil), err1), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(ctx2, a2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs((*testdata.Struct)(nil), err2), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(ctx3, a3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs((*testdata.Struct)(nil), err3), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() can handle tpp.Arg() injection", func(t *testing.T) {
		// tpp.Arg() can be used to signify that the value will be filled in later
		// by the meta-test.
		ee := []tpp.Expect{
			tpp.Given(tpp.Arg(), a1).Return(nil, err1),
			tpp.Given(tpp.Arg(), a2).Return(nil, err2),
			tpp.Given(tpp.Arg(), a3).Return(nil, err3),
		}

		ctx := context.TODO()
		m := testdata.NewMockStructyThing(_t)

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(ctx, tpp.Arg())
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(ctx, a1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs((*testdata.Struct)(nil), err1), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(ctx, a2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs((*testdata.Struct)(nil), err2), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(ctx, a3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs((*testdata.Struct)(nil), err3), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(ctx1, a1).Return(r1, nil),
			tpp.Given(ctx2, a2).Return(r2, nil),
			tpp.Given(ctx3, a3).Return(r3, nil),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Unexpected() unsets mock", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Unexpected(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(ctx1, a1).Return(r1, nil).Once(),
			tpp.Given(ctx2, a2).Return(r2, nil).Once(),
			tpp.Given(ctx3, a3).Return(r3, nil).Once(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.Equal(t, 1, c.Repeatability)
		}
	})

	t.Run("Times() sets repeatability", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		ee := []tpp.Expect{
			tpp.Given(ctx1, a1).Return(r1, nil).Times(1),
			tpp.Given(ctx2, a2).Return(r2, nil).Times(2),
			tpp.Given(ctx3, a3).Return(r3, nil).Times(3),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for i, c := range m.ExpectedCalls {
			require.Equal(t, i+1, c.Repeatability)
		}
	})

	t.Run("Arg() gets replaced with mock.Anything for empty expects", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)

		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(r1, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Len(t, m.ExpectedCalls[0].Arguments, 2)
		for _, arg := range m.ExpectedCalls[0].Arguments {
			require.Equal(t, testifymock.Anything, arg)
		}
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)

		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(r1, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(
			t,
			testifymock.Arguments([]any{r1, errTest}),
			m.ExpectedCalls[0].ReturnArguments,
		)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)

		ee := []tpp.Expect{
			tpp.Return(r1, nil),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(r2, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, testifymock.Arguments([]any{r1, nil}), m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)

		ee := []tpp.Expect{
			tpp.Err(),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(r1, error(nil)))

		require.Empty(t, m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		require.True(t, ok)
	})

	t.Run("WithDefaultReturns() causes panic if wrong number of args", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		var ee []tpp.Expect

		require.Panics(t, func() {
			tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
				return m.EXPECT().DoThing(r1, nil)
			}, tpp.WithDefaultReturns(r1, r2, r3))
		})
	})
}

// These tests check the behaviour of ExpectoriseMulti when it's passed a "bare"
// not-wrapped mock.Call, such as you get from mock.On("Foo", xxx, yyy).
//
// This isn't the most common case, since we generally pass in the wrapped call
// you get from mock.EXPECT().Foo(xxx, yyy). All the more reason to test it!
func TestExpectMultiWithTestifyMock(t *testing.T) {
	t.Run("Zero value gets empty return", func(t *testing.T) {
		m := &testifymock.Mock{}
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 1)
		require.Empty(t, m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		m := &testifymock.Mock{}
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 1)
		require.True(t, isCallOptional(m.ExpectedCalls[0]))
	})

	t.Run("Empty unsets mock", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			/* No expectations */
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Return() sets up calls", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Return(123),
			tpp.Return(456),
			tpp.Return(789),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.On("Test", i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(123), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(1), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(456), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(2), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(789), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Return() calls aren't Maybe()d", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Return(123),
			tpp.Return(456),
			tpp.Return(789),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("OK() sets up calls", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.OK(123),
			tpp.OK(456),
			tpp.OK(789),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.On("Test", i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(123), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(1), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(456), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(2), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(789), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("OK() calls aren't Maybe()d", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.OK(123),
			tpp.OK(456),
			tpp.OK(789),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Err() sets up err returns", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Err(),
			tpp.Err(),
			tpp.Err(),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.On("Test", i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0), m.ExpectedCalls[0].Arguments)
		_, ok := m.ExpectedCalls[0].ReturnArguments[0].(error)
		is.True(ok)

		is.Equal(toArgs(1), m.ExpectedCalls[1].Arguments)
		_, ok = m.ExpectedCalls[1].ReturnArguments[0].(error)
		is.True(ok)

		is.Equal(toArgs(2), m.ExpectedCalls[2].Arguments)
		_, ok = m.ExpectedCalls[2].ReturnArguments[0].(error)
		is.True(ok)
	})

	t.Run("Err() returns aren't Maybe()d", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Err(),
			tpp.Err(),
			tpp.Err(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("ErrWith() sets up err returns", func(t *testing.T) {
		var (
			errOne   = errors.New("one")
			errTwo   = errors.New("two")
			errThree = errors.New("three")
		)
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.ErrWith(errOne),
			tpp.ErrWith(errTwo),
			tpp.ErrWith(errThree),
		}

		i := 0
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			defer func() { i++ }()
			return m.On("Test", i)
		})

		is := require.New(t)
		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(0), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs(errOne), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(1), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs(errTwo), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(2), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs(errThree), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("ErrWith() returns aren't Maybe()d", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.ErrWith(errors.New("one")),
			tpp.ErrWith(errors.New("two")),
			tpp.ErrWith(errors.New("three")),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Given().Return() sets up args and return", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Given(1).Return("one"),
			tpp.Given(2).Return("two"),
			tpp.Given(3).Return("three"),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(1), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs("one"), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(2), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs("two"), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(3), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs("three"), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() sets up multiple args and return", func(t *testing.T) {
		var (
			errOne   = errors.New("one")
			errTwo   = errors.New("two")
			errThree = errors.New("three")
		)
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Given(1, 2, 3).Return("one", errOne),
			tpp.Given(4, 5, 6).Return("two", errTwo),
			tpp.Given(7, 8, 9).Return("three", errThree),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg(), tpp.Arg(), tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs(1, 2, 3), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs("one", errOne), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs(4, 5, 6), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs("two", errTwo), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs(7, 8, 9), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs("three", errThree), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() can handle tpp.Arg() injection", func(t *testing.T) {
		// tpp.Arg() can be used to signify that the value will be filled in later
		// by the meta-test.
		ee := []tpp.Expect{
			tpp.Given(tpp.Arg(), 2, 3).Return("one"),
			tpp.Given(tpp.Arg(), 5, 6).Return("two"),
			tpp.Given(tpp.Arg(), 8, 9).Return("three"),
		}

		m := &testifymock.Mock{}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", "injected", tpp.Arg(), tpp.Arg())
		})

		is := require.New(t)

		is.Len(m.ExpectedCalls, 3)

		is.Equal(toArgs("injected", 2, 3), m.ExpectedCalls[0].Arguments)
		is.Equal(toArgs("one"), m.ExpectedCalls[0].ReturnArguments)

		is.Equal(toArgs("injected", 5, 6), m.ExpectedCalls[1].Arguments)
		is.Equal(toArgs("two"), m.ExpectedCalls[1].ReturnArguments)

		is.Equal(toArgs("injected", 8, 9), m.ExpectedCalls[2].Arguments)
		is.Equal(toArgs("three"), m.ExpectedCalls[2].ReturnArguments)
	})

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Given(1).Return("one"),
			tpp.Given(2).Return("two"),
			tpp.Given(3).Return("three"),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.False(t, isCallOptional(c))
		}
	})

	t.Run("Unexpected() unsets mock", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Unexpected(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Given(1).Return("one").Once(),
			tpp.Given(2).Return("two").Once(),
			tpp.Given(3).Return("three").Once(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for _, c := range m.ExpectedCalls {
			require.Equal(t, 1, c.Repeatability)
		}
	})

	t.Run("Times() sets repeatability", func(t *testing.T) {
		m := &testifymock.Mock{}
		ee := []tpp.Expect{
			tpp.Given(1).Return("one").Times(1),
			tpp.Given(2).Return("two").Times(2),
			tpp.Given(3).Return("three").Times(3),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for i, c := range m.ExpectedCalls {
			require.Equal(t, i+1, c.Repeatability)
		}
	})

	t.Run("Arg() gets replaced with mock.Anything for empty expects", func(t *testing.T) {
		m := &testifymock.Mock{}

		var ee []tpp.Expect
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		}, tpp.WithDefaultReturns(1, 2, 3))

		require.Len(t, m.ExpectedCalls, 1)
		require.Len(t, m.ExpectedCalls[0].Arguments, 1)
		require.Equal(t, testifymock.Anything, m.ExpectedCalls[0].Arguments[0])
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := &testifymock.Mock{}

		var ee []tpp.Expect
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		}, tpp.WithDefaultReturns(1, 2, 3))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, testifymock.Arguments([]any{1, 2, 3}), m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty", func(t *testing.T) {
		m := &testifymock.Mock{}

		ee := []tpp.Expect{
			tpp.Given(1).Return("one").Times(1),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		}, tpp.WithDefaultReturns("two"))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, testifymock.Arguments([]any{"one"}), m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		m := &testifymock.Mock{}

		ee := []tpp.Expect{
			tpp.Err(),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		}, tpp.WithDefaultReturns(error(nil)))

		require.Len(t, m.ExpectedCalls, 1)
		_, ok := m.ExpectedCalls[0].ReturnArguments[0].(error)
		require.True(t, ok)
	})
}

type mockImpl struct {
	testifymock.Mock
}

func (m *mockImpl) DoSomething(x int) bool {
	args := m.Called(x)
	return args.Bool(0)
}

// There are some subtle interactions around unsetting wrapped mocks.
func TestUnexpected(t *testing.T) {
	is := require.New(t)
	mockObj := new(mockImpl)

	// Create an argument matcher
	isEven := func(x int) bool {
		return x%2 == 0
	}
	argMatcher := testifymock.MatchedBy(isEven)

	t.Run("Unsets a call with an argument matcher", func(t *testing.T) {
		call := mockObj.On("DoSomething", argMatcher).Return(true)

		unexpected := tpp.Unexpected()
		unexpected.Expectorise(call)

		mockObj.AssertExpectations(t)
		is.Empty(mockObj.ExpectedCalls)
	})

	t.Run("Unsets a wrapped call with an argument matcher", func(t *testing.T) {
		call := mockObj.On("DoSomething", argMatcher).Return(true)

		// WrappedMockCallObject is a wrapper around a mock.Call, which resembles what
		// we get from mockery.
		type WrappedMockCallObject struct {
			*testifymock.Call
		}

		fm := WrappedMockCallObject{call}

		unexpected := tpp.Unexpected()
		unexpected.Expectorise(fm)

		mockObj.AssertExpectations(t)
		is.Empty(mockObj.ExpectedCalls)
	})

	t.Run("Unsets a call", func(t *testing.T) {
		call := mockObj.On("DoSomething", 42).Return(true)

		unexpected := tpp.Unexpected()
		unexpected.Expectorise(call)

		mockObj.AssertExpectations(t)
		is.Empty(mockObj.ExpectedCalls)
	})
}

var errTest = errors.New("TEST")

func isCallOptional(call tpp.MockCall) bool {
	v := reflect.ValueOf(call)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName("optional")
	if !f.IsValid() {
		panic("no such field: optional")
	}

	// Bypass access restrictions
	ptr := unsafe.Pointer(f.UnsafeAddr())
	return *(*bool)(ptr)
}

func toArgs(a ...any) testifymock.Arguments {
	if len(a) == 0 {
		return testifymock.Arguments{}
	}
	return testifymock.Arguments(a)
}

// placeholders returns an n tpp.Arg() slice.
func placeholders(n int) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = tpp.Arg()
	}
	return out
}

// anythings returns an n testifymock.Anything slice.
func anythings(n int) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = testifymock.Anything
	}
	return out
}

func replaceLast(a []any, b any) []any {
	if len(a) == 0 {
		return a
	}
	a[len(a)-1] = b
	return a
}

func replaceAllButLast(a []any, b any) []any {
	if len(a) == 0 {
		return a
	}
	for i := 0; i < len(a)-1; i++ {
		a[i] = b
	}
	return a
}

// okReturns facilitates testing tpp.OK() by returning the given return values
// according to two filters:
//  1. The returns, but with all error types removed
//  2. The returns, but with all error types zero-valued
func okReturns(returns []any) ([]any, []any) {
	var returnsWithoutErr []any
	var returnsWithErroredNils []any
	for _, r := range returns {
		if _, ok := r.(error); ok {
			returnsWithErroredNils = append(returnsWithErroredNils, error(nil))
			continue
		}
		returnsWithoutErr = append(returnsWithoutErr, r)
		returnsWithErroredNils = append(returnsWithErroredNils, r)
	}
	return returnsWithoutErr, returnsWithErroredNils
}

func must(b bool, reason string) {
	if !b {
		panic(fmt.Sprintf("must: %s", reason))
	}
}

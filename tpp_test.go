package tpp_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/pkg/errors"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattavos/tpp"
	"github.com/mattavos/tpp/testdata"
)

// We use this dummy testing.T to pass into the code under test. We're testing
// test code, so we don't want the failed expectations within the code we're
// testing to fail *our* tests!
var _t = func() *testing.T {
	return &testing.T{}
}

// These testStructures form the parameters of various other tests here. Each
// defines a different type of mock call being configured by tpp. We then run
// tests generically over each of these structures.
//
// This might seem overly complex, but we're in the business of writing code for
// configuring mocks here, and the vast variety of mocks means that writing non-
// generic tests would be far too repetitive (believe me, I've tried it).
var testStructures = []struct {
	name            string
	argTypes        []string
	defaultArgs     []any
	returnTypes     []string
	defaultReturns  []any
	examples        []exampleCall
	expectoriseCall func(
		e tpp.Expect,
		args []any,
		opts ...tpp.ExpectoriseOption,
	) (*testifymock.Call, *testifymock.Mock)
	expectoriseMulti func(
		ee []tpp.Expect,
		args []any,
		opts ...tpp.ExpectoriseOption,
	) *testifymock.Mock
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
		expectoriseCall: func(expect tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock) {
			must(len(args) == 2, "expectoriseCall: wrong arg count")
			mock := testdata.NewMockIntyThing(_t())
			call := mock.EXPECT().DoThing(args[0], args[0])
			expect.Expectorise(call, opts...)
			return call.Call, &mock.Mock
		},
		expectoriseMulti: func(expects []tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) *testifymock.Mock {
			must(len(args) == 2, "expectoriseMulti: wrong arg count")
			mock := testdata.NewMockIntyThing(_t())
			tpp.ExpectoriseMulti(
				expects,
				func() tpp.MockCall {
					return mock.EXPECT().DoThing(args[0], args[0])
				},
				opts...,
			)
			return &mock.Mock
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
		expectoriseCall: func(expect tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock) {
			must(len(args) == 2, "expectoriseCall: wrong arg count")
			mock := testdata.NewMockStructyThing(_t())
			call := mock.EXPECT().DoThing(args[0], args[1])
			expect.Expectorise(call, opts...)
			return call.Call, &mock.Mock
		},
		expectoriseMulti: func(expects []tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) *testifymock.Mock {
			must(len(args) == 2, "expectoriseMulti: wrong arg count")
			mock := testdata.NewMockStructyThing(_t())
			tpp.ExpectoriseMulti(expects, func() tpp.MockCall {
				return mock.EXPECT().DoThing(args[0], args[1])
			}, opts...)
			return &mock.Mock
		},
	},
	{
		name:           "func(context.Context, Struct) (Struct, error)",
		argTypes:       []string{"Context", "Struct"},
		defaultArgs:    []any{context.Background(), testdata.Struct{A: 1, B: 2}},
		returnTypes:    []string{"Struct", "error"},
		defaultReturns: []any{testdata.Struct{A: 1, B: 2}, error(nil)},
		examples: []exampleCall{
			{
				name:    "ok",
				args:    []any{context.Background(), testdata.Struct{A: 1, B: 2}},
				returns: []any{testdata.Struct{A: 1, B: 2}, error(nil)},
			},
			{
				name:    "err",
				args:    []any{context.Background(), testdata.Struct{A: 3, B: 4}},
				returns: []any{testdata.Struct{}, errTest},
			},
		},
		expectoriseCall: func(expect tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock) {
			must(len(args) == 2, "expectoriseCall: wrong arg count")
			mock := testdata.NewMockConcreteStructyThing(_t())
			call := mock.EXPECT().DoThing(args[0], args[1])
			expect.Expectorise(call, opts...)
			return call.Call, &mock.Mock
		},
		expectoriseMulti: func(expects []tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) *testifymock.Mock {
			must(len(args) == 2, "expectoriseMulti: wrong arg count")
			mock := testdata.NewMockConcreteStructyThing(_t())
			tpp.ExpectoriseMulti(expects, func() tpp.MockCall {
				return mock.EXPECT().DoThing(args[0], args[1])
			}, opts...)
			return &mock.Mock
		},
	},
	{
		name:           "func(context.Context, []int) ([]int, error)",
		argTypes:       []string{"Context", "[]int"},
		defaultArgs:    []any{context.Background(), []int{1, 2, 3}},
		returnTypes:    []string{"[]int", "error"},
		defaultReturns: []any{[]int{4, 5, 6}, error(nil)},
		examples: []exampleCall{
			{
				name:    "ok",
				args:    []any{context.Background(), []int{1, 2, 3}},
				returns: []any{[]int{4, 5, 6}, error(nil)},
			},
			{
				name:    "empty",
				args:    []any{context.Background(), []int{1, 2, 3}},
				returns: []any{[]int{}, error(nil)},
			},
			{
				name:    "err",
				args:    []any{context.Background(), []int{1}},
				returns: []any{([]int)(nil), errTest},
			},
		},
		expectoriseCall: func(expect tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock) {
			must(len(args) == 2, "expectoriseCall: wrong arg count")
			mock := testdata.NewMockSliceyThing(_t())
			call := mock.EXPECT().DoThing(args[0], args[1])
			expect.Expectorise(call, opts...)
			return call.Call, &mock.Mock
		},
		expectoriseMulti: func(expects []tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) *testifymock.Mock {
			must(len(args) == 2, "expectoriseMulti: wrong arg count")
			mock := testdata.NewMockSliceyThing(_t())
			tpp.ExpectoriseMulti(expects, func() tpp.MockCall {
				return mock.EXPECT().DoThing(args[0], args[1])
			}, opts...)
			return &mock.Mock
		},
	},
	{
		name:           "func()",
		argTypes:       []string{},
		defaultArgs:    []any{},
		returnTypes:    []string{},
		defaultReturns: []any{},
		examples: []exampleCall{
			{
				name:    "empty",
				args:    []any{},
				returns: []any{},
			},
		},
		expectoriseCall: func(expect tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock) {
			must(len(args) == 0, "expectoriseCall: wrong arg count")
			mock := testdata.NewMockEmptyThing(_t())
			call := mock.EXPECT().DoThing().Return()
			expect.Expectorise(call, opts...)
			return call.Call, &mock.Mock
		},
		expectoriseMulti: func(expects []tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) *testifymock.Mock {
			must(len(args) == 0, "expectoriseMulti: wrong arg count")
			mock := testdata.NewMockEmptyThing(_t())
			tpp.ExpectoriseMulti(
				expects,
				func() tpp.MockCall {
					return mock.EXPECT().DoThing()
				},
				opts...,
			)
			return &mock.Mock
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
		expectoriseCall: func(expect tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) (*testifymock.Call, *testifymock.Mock) {
			mock := &testifymock.Mock{}
			call := mock.On("Test", args...)
			expect.Expectorise(call, opts...)
			return call, mock
		},
		expectoriseMulti: func(expects []tpp.Expect, args []any, opts ...tpp.ExpectoriseOption) *testifymock.Mock {
			mock := &testifymock.Mock{}
			tpp.ExpectoriseMulti(
				expects,
				func() tpp.MockCall {
					return mock.On("Test", args...)
				},
				opts...,
			)
			return mock
		},
	},
}

func TestExpect(t *testing.T) {
	for _, tt := range testStructures {
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
					requireEqualArgs(t, example.returns, call.ReturnArguments)
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
						requireEqualArgs(t, returnsWithNilledErrs, call.ReturnArguments)
					})

					t.Run("OK() is not Maybe()d: "+example.name, func(t *testing.T) {
						call, _ := tt.expectoriseCall(
							tpp.OK(returnsWithoutErrTypes...),
							tt.defaultArgs,
						)
						require.False(t, isCallOptional(call))
					})
				}

				if contains(tt.returnTypes, "error") {
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

						requireEqualArgs(t, example.args, c.Arguments)
						requireEqualArgs(t, example.returns, c.ReturnArguments)
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

						requireEqualArgs(t, args, call.Arguments)
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

						requireEqualArgs(t, example.args, call.Arguments)
						requireEqualArgs(t, example.returns, call.ReturnArguments)
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
				requireEqualArgs(t, tt.defaultReturns, call.ReturnArguments)
			})

			t.Run("Injecting() works with zero value Expect", func(t *testing.T) {
				var expect tpp.Expect
				for _, ret := range tt.defaultReturns {
					expect = *expect.Injecting(ret)
				}
				call, _ := tt.expectoriseCall(expect, tt.defaultArgs)
				requireEqualArgs(t, tt.defaultReturns, call.ReturnArguments)
			})

			t.Run("Arg() gets replaced with mock.Anything for empty expects", func(t *testing.T) {
				var expect tpp.Expect
				call, _ := tt.expectoriseCall(expect, placeholders(len(tt.defaultArgs)))
				requireEqualArgs(t, anythings(len(tt.defaultArgs)), call.Arguments)
			})

			t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
				var expect tpp.Expect
				call, _ := tt.expectoriseCall(
					expect,
					tt.defaultArgs,
					tpp.WithDefaultReturns(tt.defaultReturns...),
				)
				requireEqualArgs(t, tt.defaultReturns, call.ReturnArguments)
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
					requireEqualArgs(t, tt.defaultReturns, call.ReturnArguments)
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
					wrongArgs := make([]wrong, len(tt.defaultReturns))
					for i := range tt.defaultReturns {
						wrongArgs[i] = wrong{}
					}

					var expect tpp.Expect

					require.Panics(t, func() {
						tt.expectoriseCall(
							expect,
							tt.defaultArgs,
							tpp.WithDefaultReturns(wrongArgs),
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

func TestExpectMulti(t *testing.T) {
	for _, tt := range testStructures {
		t.Run(tt.name, func(t *testing.T) {

			t.Run("Zero value gets empty return", func(t *testing.T) {
				var expects []tpp.Expect
				mock := tt.expectoriseMulti(expects, tt.defaultArgs)
				require.Len(t, mock.ExpectedCalls, 1)
				for _, a := range mock.ExpectedCalls[0].ReturnArguments {
					require.Empty(t, a)
				}
			})

			t.Run("Zero value is Maybe()d", func(t *testing.T) {
				var expects []tpp.Expect
				mock := tt.expectoriseMulti(expects, tt.defaultArgs)
				require.Len(t, mock.ExpectedCalls, 1)
				require.True(t, isCallOptional(mock.ExpectedCalls[0]))
			})

			t.Run("Empty expectations unsets mock", func(t *testing.T) {
				expects := []tpp.Expect{
					/* No expectations */
				}
				mock := tt.expectoriseMulti(expects, tt.defaultArgs)
				require.Empty(t, mock.ExpectedCalls)
			})

			t.Run("Return() sets up calls", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Return(tt.defaultReturns...),
					tpp.Return(tt.defaultReturns...),
					tpp.Return(tt.defaultReturns...),
				}
				mock := tt.expectoriseMulti(expects, tt.defaultArgs)

				require.Len(t, mock.ExpectedCalls, 3)
				for _, call := range mock.ExpectedCalls {
					requireEqualArgs(t, tt.defaultReturns, call.ReturnArguments)
				}
			})

			t.Run("Return() calls aren't Maybe()d", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Return(tt.defaultReturns...),
					tpp.Return(tt.defaultReturns...),
					tpp.Return(tt.defaultReturns...),
				}
				mock := tt.expectoriseMulti(expects, tt.defaultArgs)

				require.Len(t, mock.ExpectedCalls, 3)
				for _, call := range mock.ExpectedCalls {
					require.False(t, isCallOptional(call))
				}
			})

			{
				// tpp.OK() will automagically nil out error values in the return, so
				// we need a version of the return example without errors specified.
				returnsWithoutErrTypes, returnsWithNilledErrs := okReturns(tt.defaultReturns)

				t.Run("OK() sets up return", func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.OK(returnsWithoutErrTypes...),
						tpp.OK(returnsWithoutErrTypes...),
						tpp.OK(returnsWithoutErrTypes...),
					}
					mock := tt.expectoriseMulti(expects, tt.defaultArgs)

					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						requireEqualArgs(t, returnsWithNilledErrs, call.ReturnArguments)
					}
				})

				t.Run("OK() is not Maybe()d", func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.OK(returnsWithoutErrTypes...),
						tpp.OK(returnsWithoutErrTypes...),
						tpp.OK(returnsWithoutErrTypes...),
					}
					mock := tt.expectoriseMulti(expects, tt.defaultArgs)

					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						require.False(t, isCallOptional(call))
					}
				})
			}

			if contains(tt.returnTypes, "error") {
				t.Run("Err() sets up err return", func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.Err(),
						tpp.Err(),
						tpp.Err(),
					}
					mock := tt.expectoriseMulti(expects, tt.defaultArgs)

					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						for i, ret := range tt.returnTypes {
							if ret == "error" {
								_, ok := call.ReturnArguments[i].(error)
								require.True(t, ok)
							} else {
								require.Empty(t, call.ReturnArguments[i])
							}
						}
					}
				})

				t.Run("Err() is not Maybe()d", func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.Err(),
						tpp.Err(),
						tpp.Err(),
					}
					mock := tt.expectoriseMulti(expects, tt.defaultArgs)
					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						require.False(t, isCallOptional(call))
					}
				})

				t.Run("ErrWith() sets up err return", func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.ErrWith(errTest),
						tpp.ErrWith(errTest),
						tpp.ErrWith(errTest),
					}
					mock := tt.expectoriseMulti(expects, tt.defaultArgs)

					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						for i, ret := range tt.returnTypes {
							if ret == "error" {
								rerr, ok := call.ReturnArguments[i].(error)
								require.True(t, ok)
								require.Equal(t, errTest, rerr)
							} else {
								require.Empty(t, call.ReturnArguments[i])
							}
						}
					}
				})

				t.Run("ErrWith() is not Maybe()d", func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.ErrWith(errTest),
						tpp.ErrWith(errTest),
						tpp.ErrWith(errTest),
					}
					mock := tt.expectoriseMulti(expects, tt.defaultArgs)
					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						require.False(t, isCallOptional(call))
					}
				})
			}

			t.Run("Given().Return() sets up calls", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
				}

				mock := tt.expectoriseMulti(expects, placeholders(len(tt.defaultArgs)))

				require.Len(t, mock.ExpectedCalls, 3)
				for _, call := range mock.ExpectedCalls {
					requireEqualArgs(t, tt.defaultArgs, call.Arguments)
					requireEqualArgs(t, tt.defaultReturns, call.ReturnArguments)
				}
			})

			for _, example := range tt.examples {
				t.Run("Given().Return() sets up calls: example:"+example.name, func(t *testing.T) {
					expects := []tpp.Expect{
						tpp.Given(example.args...).Return(example.returns...),
						tpp.Given(example.args...).Return(example.returns...),
						tpp.Given(example.args...).Return(example.returns...),
					}

					mock := tt.expectoriseMulti(expects, placeholders(len(example.args)))

					require.Len(t, mock.ExpectedCalls, 3)
					for _, call := range mock.ExpectedCalls {
						requireEqualArgs(t, example.args, call.Arguments)
						requireEqualArgs(t, example.returns, call.ReturnArguments)
					}
				},
				)

				t.Run(
					"Given().Return() can handle tpp.Arg() injection: example:"+example.name,
					func(t *testing.T) {
						// tpp.Arg() can be used to signify that the value will be filled in
						// later by the meta-test. So here we Expect using tpp.Arg() as the
						// final arg, and then when we call the mock we fill that value in,
						// and pass the other args as tpp.Arg() (since they *were* specified
						// by the Expect).
						expectArgs := replaceLast(example.args, tpp.Arg())
						callArgs := replaceAllButLast(example.args, tpp.Arg())

						expects := []tpp.Expect{
							tpp.Given(expectArgs...).Return(example.returns...),
							tpp.Given(example.args...).Return(example.returns...),
							tpp.Given(example.args...).Return(example.returns...),
						}
						mock := tt.expectoriseMulti(expects, callArgs)

						require.Len(t, mock.ExpectedCalls, 3)
						for _, call := range mock.ExpectedCalls {
							requireEqualArgs(t, example.args, call.Arguments)
							requireEqualArgs(t, example.returns, call.ReturnArguments)
						}
					},
				)
			}

			t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
				}

				mock := tt.expectoriseMulti(expects, placeholders(len(tt.defaultArgs)))

				require.Len(t, mock.ExpectedCalls, 3)
				for _, call := range mock.ExpectedCalls {
					require.False(t, isCallOptional(call))
				}
			})

			t.Run("Unexpected() unsets mock", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Unexpected(),
				}
				mock := tt.expectoriseMulti(expects, placeholders(len(tt.defaultArgs)))
				require.Empty(t, mock.ExpectedCalls)
			})

			t.Run("Once() sets repeatability", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Once(),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Once(),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Once(),
				}
				mock := tt.expectoriseMulti(expects, placeholders(len(tt.defaultArgs)))
				require.Len(t, mock.ExpectedCalls, 3)
				for _, call := range mock.ExpectedCalls {
					require.Equal(t, 1, call.Repeatability)
				}
			})

			t.Run("Times() sets repeatability", func(t *testing.T) {
				expects := []tpp.Expect{
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Times(1),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Times(2),
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...).Times(3),
				}
				mock := tt.expectoriseMulti(expects, placeholders(len(tt.defaultArgs)))
				require.Len(t, mock.ExpectedCalls, 3)
				for i, call := range mock.ExpectedCalls {
					require.Equal(t, i+1, call.Repeatability)
				}
			})

			t.Run("Arg() gets replaced with mock.Anything for empty expects", func(t *testing.T) {
				var expects []tpp.Expect
				mock := tt.expectoriseMulti(expects, placeholders(len(tt.defaultArgs)))

				require.Len(t, mock.ExpectedCalls, 1)
				require.Len(t, mock.ExpectedCalls[0].Arguments, len(tt.defaultArgs))
				for _, arg := range mock.ExpectedCalls[0].Arguments {
					require.Equal(t, testifymock.Anything, arg)
				}
			})

			t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
				var expects []tpp.Expect
				mock := tt.expectoriseMulti(
					expects,
					placeholders(len(tt.defaultArgs)),
					tpp.WithDefaultReturns(tt.defaultReturns...),
				)

				require.Len(t, mock.ExpectedCalls, 1)
				requireEqualArgs(t, tt.defaultReturns, mock.ExpectedCalls[0].ReturnArguments)
			})

			t.Run("WithDefaultReturns() adds to return only if Expect empty", func(t *testing.T) {
				type wrong struct{}

				expects := []tpp.Expect{
					tpp.Given(tt.defaultArgs...).Return(tt.defaultReturns...),
				}
				mock := tt.expectoriseMulti(
					expects,
					placeholders(len(tt.defaultArgs)),
					tpp.WithDefaultReturns(wrong{}),
				)

				require.Len(t, mock.ExpectedCalls, 1)
				requireEqualArgs(t, tt.defaultReturns, mock.ExpectedCalls[0].ReturnArguments)
			})

			if len(tt.returnTypes) > 0 {
				t.Run(
					"WithDefaultReturns() causes panic if wrong number of args",
					func(t *testing.T) {
						var wrongArgs []any
						for _, a := range tt.defaultReturns {
							// Double sized
							wrongArgs = append(wrongArgs, a)
							wrongArgs = append(wrongArgs, a)
						}

						var expects []tpp.Expect
						require.Panics(t, func() {
							tt.expectoriseMulti(
								expects,
								placeholders(len(tt.defaultArgs)),
								tpp.WithDefaultReturns(wrongArgs),
							)
						})
					},
				)

				t.Run(
					"WithDefaultReturns() causes panic if wrong type of args",
					func(t *testing.T) {
						type wrong struct{}
						wrongArgs := make([]wrong, len(tt.defaultReturns))
						for i := range tt.defaultReturns {
							wrongArgs[i] = wrong{}
						}

						var expects []tpp.Expect
						require.Panics(t, func() {
							tt.expectoriseMulti(
								expects,
								placeholders(len(tt.defaultArgs)),
								tpp.WithDefaultReturns(wrongArgs),
							)
						})
					},
				)
			}

		})
	}
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

type exampleCall struct {
	name    string
	args    []any
	returns []any
}

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

// requireEqualArgs compares two mock.Arguments (either args or returns).
func requireEqualArgs(t *testing.T, a []any, b testifymock.Arguments) {
	require.Equal(t, len(a), len(b))

	if len(a) == 0 {
		// Testify is pretty inconsistent with whether it uses nil or empty args,
		// so checking for == 0 is the best we can do.
		return
	}

	require.Equal(t, testifymock.Arguments(a), b)
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

// contains reports whether v is present in s.
func contains[S ~[]E, E comparable](s S, v E) bool {
	for i := range s {
		if v == s[i] {
			return true
		}
	}
	return false
}

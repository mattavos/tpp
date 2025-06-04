package tpp_test

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattavos/tpp"
	"github.com/mattavos/tpp/testdata"
)

// We're testing using a mockery mock of an interface which looks like this:
//
//	type IntyThing interface {
//		DoThing(a, b int) (int, error)
//	}
func TestExpectWithMockeryMockIntyThing(t *testing.T) {
	_t := &testing.T{} // dummy testing.T for passing into code under test

	t.Run("Zero value gets empty return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		var e tpp.Expect
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 2)
		for _, a := range c.ReturnArguments {
			require.Empty(t, a)
		}
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		var e tpp.Expect
		e.Expectorise(c)

		require.True(t, isCallOptional(c))
	})

	t.Run("Return() setups up return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.Return(123, errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs(123, errTest), c.ReturnArguments)
	})

	t.Run("Return() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.Return(123, errTest)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("OK() setups up return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.OK(123)
		e.Expectorise(c)

		require.Equal(t, toArgs(123, error(nil)), c.ReturnArguments)
	})

	t.Run("OK() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.OK(123)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Err() setups up err return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.Err()
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 2)
		require.Empty(t, c.ReturnArguments[0])
		_, ok := c.ReturnArguments[1].(error)
		require.True(t, ok)
	})

	t.Run("Err() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.Err()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("ErrWith() setups up err return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.ErrWith(errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs(0, errTest), c.ReturnArguments)
	})

	t.Run("ErrWith() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.ErrWith(errTest)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Given().Return() setups up args and return: no err", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(123, 456).Return(789, error(nil))
		e.Expectorise(c)

		require.Equal(t, toArgs(123, 456), c.Arguments)
		require.Equal(t, toArgs(789, error(nil)), c.ReturnArguments)
	})

	t.Run("Given().Return() setups up args and return: err", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(123, 456).Return(789, errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs(123, 456), c.Arguments)
		require.Equal(t, toArgs(789, errTest), c.ReturnArguments)
	})

	t.Run("Given().Return() setups up args with mock.Anything", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(123, mock.Anything).Return(789, error(nil))
		e.Expectorise(c)

		require.Equal(t, toArgs(123, mock.Anything), c.Arguments)
	})

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(123, 456).Return(789, error(nil))
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Unexpected() unsets mock", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Unexpected()
		e.Expectorise(c)

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.OK(123).Once()
		e.Expectorise(c)

		require.Equal(t, 1, c.Repeatability)
	})

	t.Run("Times() sets repeatability", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.OK(123).Times(42)
		e.Expectorise(c)

		require.Equal(t, 42, c.Repeatability)
	})

	t.Run("Injecting() adds to return", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.OK( /* provided by injection */ )
		e.Injecting(123).Expectorise(c)

		require.Equal(t, toArgs(123, error(nil)), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		var e tpp.Expect
		e.Expectorise(c, tpp.WithDefaultReturns(123, errTest))

		require.Equal(t, toArgs(123, errTest), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Return)", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.Return(123, errTest)
		e.Expectorise(c, tpp.WithDefaultReturns(456, error(nil)))

		require.Equal(t, toArgs(123, errTest), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		e := tpp.Err()
		e.Expectorise(c, tpp.WithDefaultReturns(456, error(nil)))

		require.Empty(t, m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		require.True(t, ok)
	})

	t.Run("WithDefaultReturns() causes panic if wrong number of args", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2)

		var e tpp.Expect

		require.Panics(t, func() {
			e.Expectorise(c, tpp.WithDefaultReturns(1, 2, 3, 5, error(nil)))
		})
	})

	t.Run("WithDefaultReturns() causes panic if wrong type args", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)
		c := m.EXPECT().DoThing(1, 2) // returns int, error

		var e tpp.Expect

		require.Panics(t, func() {
			e.Expectorise(c, tpp.WithDefaultReturns("wrong", "types", error(nil)))
		})
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

	t.Run("Return() setups up calls", func(t *testing.T) {
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

	t.Run("OK() setups up calls", func(t *testing.T) {
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

	t.Run("Err() setups up err returns", func(t *testing.T) {
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

	t.Run("ErrWith() setups up err returns", func(t *testing.T) {
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

	t.Run("Given().Return() setups up args and return", func(t *testing.T) {
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

	t.Run("Given().Return() setups up args and return: errors", func(t *testing.T) {
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
			tpp.Given(1, 1).Return(1).Once(),
			tpp.Given(2, 2).Return(2).Once(),
			tpp.Given(3, 3).Return(3).Once(),
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
			tpp.Given(1, 1).Return(1).Times(1),
			tpp.Given(2, 2).Return(2).Times(2),
			tpp.Given(3, 3).Return(3).Times(3),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		})

		require.Len(t, m.ExpectedCalls, 3)
		for i, c := range m.ExpectedCalls {
			require.Equal(t, i+1, c.Repeatability)
		}
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := testdata.NewMockIntyThing(_t)

		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(1, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, mock.Arguments([]any{1, errTest}), m.ExpectedCalls[0].ReturnArguments)
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
		require.Equal(t, mock.Arguments([]any{123, errTest}), m.ExpectedCalls[0].ReturnArguments)
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
func TestExpectWithMockeryMockStructyThing(t *testing.T) {
	var (
		_t = &testing.T{} // dummy testing.T for passing into code under test
		s1 = &testdata.Struct{A: 1, B: 2}
		s2 = &testdata.Struct{A: 10, B: 20}
	)

	t.Run("Zero value gets empty return", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		var e tpp.Expect
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 2)
		for _, a := range c.ReturnArguments {
			require.Empty(t, a)
		}
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		var e tpp.Expect
		e.Expectorise(c)

		require.True(t, isCallOptional(c))
	})

	t.Run("Return() setups up return", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Return(s2, errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs(s2, errTest), c.ReturnArguments)
	})

	t.Run("Return() is not Maybe()d", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Return(s2, errTest)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("OK() setups up return", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.OK(s2)
		e.Expectorise(c)

		require.Equal(t, toArgs(s2, error(nil)), c.ReturnArguments)
	})

	t.Run("OK() is not Maybe()d", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.OK(s2)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Err() setups up err return", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Err()
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 2)
		require.Empty(t, c.ReturnArguments[0])
		_, ok := c.ReturnArguments[1].(error)
		require.True(t, ok)
	})

	t.Run("Err() is not Maybe()d", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Err()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("ErrWith() setups up err return", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.ErrWith(errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs((*testdata.Struct)(nil), errTest), c.ReturnArguments)
	})

	t.Run("ErrWith() is not Maybe()d", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.ErrWith(errTest)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Given().Return() setups up args and return: no err", func(t *testing.T) {
		ctx := context.TODO()
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(ctx, s1).Return(s2, nil)
		e.Expectorise(c)

		require.Equal(t, toArgs(ctx, s1), c.Arguments)
		require.Equal(t, toArgs(s2, error(nil)), c.ReturnArguments)
	})

	t.Run("Given().Return() setups up args and return: err", func(t *testing.T) {
		ctx := context.TODO()
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(ctx, s1).Return(nil, errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs(ctx, s1), c.Arguments)
		require.Equal(t, toArgs((*testdata.Struct)(nil), errTest), c.ReturnArguments)
	})

	t.Run("Given().Return() setups up args with mock.Anything", func(t *testing.T) {
		ctx := context.TODO()
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(ctx, mock.Anything).Return(s2, nil)
		e.Expectorise(c)

		require.Equal(t, toArgs(ctx, mock.Anything), c.Arguments)
	})

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(tpp.Arg(), tpp.Arg())

		e := tpp.Given(context.TODO(), s1).Return(s2, nil)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Unexpected() unsets mock", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		c := m.EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Unexpected()
		e.Expectorise(c)

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Return(s1, nil).Once()
		e.Expectorise(c)

		require.Equal(t, 1, c.Repeatability)
	})

	t.Run("Times() sets repeatability", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Return(s2, nil).Times(42)
		e.Expectorise(c)

		require.Equal(t, 42, c.Repeatability)
	})

	t.Run("Injecting() adds to return", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.OK( /* provided by injection */ )
		e.Injecting(s2).Expectorise(c)

		require.Equal(t, toArgs(s2, error(nil)), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		var e tpp.Expect
		e.Expectorise(c, tpp.WithDefaultReturns(s2, errTest))

		require.Equal(t, toArgs(s2, errTest), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Return)", func(t *testing.T) {
		c := testdata.NewMockStructyThing(_t).
			EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Return(s2, nil)
		e.Expectorise(c, tpp.WithDefaultReturns(s1, errTest))

		require.Equal(t, toArgs(s2, nil), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		c := m.EXPECT().DoThing(context.TODO(), s1)

		e := tpp.Err()
		e.Expectorise(c, tpp.WithDefaultReturns(s2, error(nil)))

		require.Empty(t, m.ExpectedCalls[0].ReturnArguments[0])
		_, ok := m.ExpectedCalls[0].ReturnArguments[1].(error)
		require.True(t, ok)
	})

	t.Run("WithDefaultReturns() causes panic if wrong number of args", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		c := m.EXPECT().DoThing(context.TODO(), s1)

		var e tpp.Expect

		require.Panics(t, func() {
			e.Expectorise(c, tpp.WithDefaultReturns(s1, s1, s1, error(nil)))
		})
	})

	t.Run("WithDefaultReturns() causes panic if wrong type args", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)
		c := m.EXPECT().DoThing(context.TODO(), s1)

		var e tpp.Expect

		require.Panics(t, func() {
			e.Expectorise(c, tpp.WithDefaultReturns("wrong", "types", error(nil)))
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

	t.Run("Return() setups up calls", func(t *testing.T) {
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

	t.Run("OK() setups up calls", func(t *testing.T) {
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

	t.Run("Err() setups up err returns", func(t *testing.T) {
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

	t.Run("ErrWith() setups up err returns", func(t *testing.T) {
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

	t.Run("Given().Return() setups up args and return", func(t *testing.T) {
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

	t.Run("Given().Return() setups up args and return: errors", func(t *testing.T) {
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

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := testdata.NewMockStructyThing(_t)

		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.EXPECT().DoThing(tpp.Arg(), tpp.Arg())
		}, tpp.WithDefaultReturns(r1, errTest))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, mock.Arguments([]any{r1, errTest}), m.ExpectedCalls[0].ReturnArguments)
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
		require.Equal(t, mock.Arguments([]any{r1, nil}), m.ExpectedCalls[0].ReturnArguments)
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

// These tests check the behaviour of Expectorise when it's passed a "bare"
// not-wrapped mock.Call, such as you get from mock.On("Foo", xxx, yyy).
//
// This isn't the most common case, since we generally pass in the wrapped call
// you get from mock.EXPECT().Foo(xxx, yyy). All the more reason to test it!
func TestExpectWithTestifyMock(t *testing.T) {
	t.Run("Zero value gets empty return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		var e tpp.Expect
		e.Expectorise(c)

		require.Empty(t, c.ReturnArguments)
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		var e tpp.Expect
		e.Expectorise(c)

		require.True(t, isCallOptional(c))
	})

	t.Run("Return() setups up return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Return(123)
		e.Expectorise(c)

		require.Equal(t, toArgs(123), c.ReturnArguments)
	})

	t.Run("Return() is not Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Return()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("OK() setups up return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.OK(123)
		e.Expectorise(c)

		require.Equal(t, toArgs(123), c.ReturnArguments)
	})

	t.Run("OK() is not Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.OK()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Err() setups up err return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Err()
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 1)
		_, ok := c.ReturnArguments[0].(error)
		require.True(t, ok)
	})

	t.Run("Err() is not Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Err()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("ErrWith() setups up err return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		withErr := errors.New("Everything exploded")
		e := tpp.ErrWith(withErr)
		e.Expectorise(c)

		require.Len(t, c.ReturnArguments, 1)
		err, ok := c.ReturnArguments[0].(error)
		require.True(t, ok)
		require.Equal(t, withErr, err)
	})

	t.Run("ErrWith() is not Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		withErr := errors.New("Everything exploded")
		e := tpp.ErrWith(withErr)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Given().Return() setups up args and return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", tpp.Arg())

		e := tpp.Given(123).Return(456)
		e.Expectorise(c)

		require.Equal(t, toArgs(123), c.Arguments)
		require.Equal(t, toArgs(456), c.ReturnArguments)
	})

	t.Run("Given().Return() setups up multiple args and return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", tpp.Arg(), tpp.Arg(), tpp.Arg())

		e := tpp.Given(1, 2, 3).Return(456, errTest)
		e.Expectorise(c)

		require.Equal(t, toArgs(1, 2, 3), c.Arguments)
		require.Equal(t, toArgs(456, errTest), c.ReturnArguments)
	})

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", tpp.Arg())

		e := tpp.Given(123).Return(456)
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Unexpected() unsets mock", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)
		require.Len(t, c.Parent.ExpectedCalls, 1)

		e := tpp.Unexpected()
		e.Expectorise(c)

		require.Empty(t, c.Parent.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.OK(123).Once()
		e.Expectorise(c)

		require.Equal(t, 1, c.Repeatability)
	})

	t.Run("Times() sets repeatability", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.OK(123).Times(42)
		e.Expectorise(c)

		require.Equal(t, 42, c.Repeatability)
	})

	t.Run("Injecting() adds to return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.OK(123)
		e.Injecting(456).Expectorise(c)

		require.Equal(t, mock.Arguments(mock.Arguments{123, 456}), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		var e tpp.Expect
		e.Expectorise(c, tpp.WithDefaultReturns(123, errTest))

		require.Equal(t, toArgs(123, errTest), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Return(123, errTest)
		e.Expectorise(c, tpp.WithDefaultReturns(456, error(nil)))

		require.Equal(t, toArgs(123, errTest), c.ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Err()
		e.Expectorise(c, tpp.WithDefaultReturns(456, error(nil)))

		_, ok := c.ReturnArguments[0].(error)
		require.True(t, ok)
	})
}

// These tests check the behaviour of ExpectoriseMulti when it's passed a "bare"
// not-wrapped mock.Call, such as you get from mock.On("Foo", xxx, yyy).
//
// This isn't the most common case, since we generally pass in the wrapped call
// you get from mock.EXPECT().Foo(xxx, yyy). All the more reason to test it!
func TestExpectMultiWithTestifyMock(t *testing.T) {
	t.Run("Zero value gets empty return", func(t *testing.T) {
		m := &mock.Mock{}
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 1)
		require.Empty(t, m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("Zero value is Maybe()d", func(t *testing.T) {
		m := &mock.Mock{}
		var ee []tpp.Expect

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", 1)
		})

		require.Len(t, m.ExpectedCalls, 1)
		require.True(t, isCallOptional(m.ExpectedCalls[0]))
	})

	t.Run("Empty unsets mock", func(t *testing.T) {
		m := &mock.Mock{}
		ee := []tpp.Expect{
			/* No expectations */
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Return() setups up calls", func(t *testing.T) {
		m := &mock.Mock{}
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
		m := &mock.Mock{}
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

	t.Run("OK() setups up calls", func(t *testing.T) {
		m := &mock.Mock{}
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
		m := &mock.Mock{}
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

	t.Run("Err() setups up err returns", func(t *testing.T) {
		m := &mock.Mock{}
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
		m := &mock.Mock{}
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

	t.Run("ErrWith() setups up err returns", func(t *testing.T) {
		var (
			errOne   = errors.New("one")
			errTwo   = errors.New("two")
			errThree = errors.New("three")
		)
		m := &mock.Mock{}
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
		m := &mock.Mock{}
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

	t.Run("Given().Return() setups up args and return", func(t *testing.T) {
		m := &mock.Mock{}
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

	t.Run("Given().Return() setups up multiple args and return", func(t *testing.T) {
		var (
			errOne   = errors.New("one")
			errTwo   = errors.New("two")
			errThree = errors.New("three")
		)
		m := &mock.Mock{}
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

	t.Run("Given().Return() is not Maybe()d", func(t *testing.T) {
		m := &mock.Mock{}
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
		m := &mock.Mock{}
		ee := []tpp.Expect{
			tpp.Unexpected(),
		}

		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		})

		require.Empty(t, m.ExpectedCalls)
	})

	t.Run("Once() sets repeatability", func(t *testing.T) {
		m := &mock.Mock{}
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
		m := &mock.Mock{}
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

	t.Run("WithDefaultReturns() adds to return if Expect empty", func(t *testing.T) {
		m := &mock.Mock{}

		var ee []tpp.Expect
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		}, tpp.WithDefaultReturns(1, 2, 3))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, mock.Arguments([]any{1, 2, 3}), m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty", func(t *testing.T) {
		m := &mock.Mock{}

		ee := []tpp.Expect{
			tpp.Given(1).Return("one").Times(1),
		}
		tpp.ExpectoriseMulti(ee, func() tpp.MockCall {
			return m.On("Test", tpp.Arg())
		}, tpp.WithDefaultReturns("two"))

		require.Len(t, m.ExpectedCalls, 1)
		require.Equal(t, mock.Arguments([]any{"one"}), m.ExpectedCalls[0].ReturnArguments)
	})

	t.Run("WithDefaultReturns() adds to return only if Expect empty (Err)", func(t *testing.T) {
		m := &mock.Mock{}

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
	mock.Mock
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
	argMatcher := mock.MatchedBy(isEven)

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
			*mock.Call
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

func toArgs(a ...any) mock.Arguments {
	return mock.Arguments(a)
}

var errTest = errors.New("TEST")

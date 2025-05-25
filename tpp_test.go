package tpp_test

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattavos/tpp"
)

// These tests check the behaviour of Expectorise when it's passed a "bare"
// not-wrapped mock.Call, such as you get from mock.On("Foo", xxx, yyy).
//
// This isn't the most common case, since we generally pass in the wrapped call
// you get from mock.EXPECT().Foo(xxx, yyy). All the more reason to test it!
func TestExpectoriseWithBareMock(t *testing.T) {
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

		require.Equal(t, mock.Arguments(mock.Arguments{123}), c.ReturnArguments)
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

		require.Equal(t, mock.Arguments(mock.Arguments{123}), c.ReturnArguments)
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

	t.Run("Err() is not Maybe()d", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1)

		e := tpp.Err()
		e.Expectorise(c)

		require.False(t, isCallOptional(c))
	})

	t.Run("Given().Return() setups up args and return", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", tpp.Arg())

		e := tpp.Given(123).Return(456)
		e.Expectorise(c)

		require.Equal(t, mock.Arguments(mock.Arguments{123}), c.Arguments)
		require.Equal(t, mock.Arguments(mock.Arguments{456}), c.ReturnArguments)
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

func isCallOptional(call *mock.Call) bool {
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

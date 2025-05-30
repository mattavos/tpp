package tpp

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	obj "github.com/mattavos/tpp/testdata"
)

// We unfortunately have to rely on reflection to get the type information for
// mock call arguments and returns. Of course, this makes us vulnerable to
// changes in testify/mockery breaking our code. These tests are a canary for
// breaking changes at this layer. If one of these tests fails, it's likely that
// something in testify/mockery has changed in an incompatible way.

func TestReflect(t *testing.T) {
	_t := &testing.T{} // dummy testing.T for passing into code under test

	t.Run("newReflectedMockCall", func(t *testing.T) {
		c := obj.NewMockObj(_t).EXPECT().DoThing(1, 2)
		rm, err := newReflectedMockCall(c)
		require.NoError(t, err)
		require.NotNil(t, rm)
	})

	t.Run("GetArguments", func(t *testing.T) {
		c := obj.NewMockObj(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)
		args, err := rm.GetArguments()

		is := require.New(t)
		is.NoError(err)
		is.Equal([]any{1, 2}, args)
	})

	t.Run("SetArguments", func(t *testing.T) {
		c := obj.NewMockObj(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)

		rm.SetArguments([]any{3, 4})

		args, err := rm.GetArguments()
		is := require.New(t)
		is.NoError(err)
		is.Equal([]any{3, 4}, args)
	})

	t.Run("CallReturnEmpty: nil err", func(t *testing.T) {
		c := obj.NewMockObj(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)
		rm.CallReturnEmpty(nil)
		// DoThing returns (int, error), so empty is 0, nil
		require.Equal(t, mock.Arguments(mock.Arguments{0, nil}), c.ReturnArguments)
	})

	t.Run("CallReturnEmpty: with err", func(t *testing.T) {
		errTest := errors.New("ERROR")
		c := obj.NewMockObj(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)
		rm.CallReturnEmpty(errTest)
		// DoThing returns (int, error)
		require.Equal(t, mock.Arguments(mock.Arguments{0, errTest}), c.ReturnArguments)
	})
}

func TestReflectWithTestifyMock(t *testing.T) {
	t.Run("newReflectedMockCall", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1, 2)
		rm, err := newReflectedMockCall(c)
		require.NoError(t, err)
		require.NotNil(t, rm)
	})

	t.Run("GetArguments", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1, 2)
		rm, _ := newReflectedMockCall(c)

		args, err := rm.GetArguments()

		is := require.New(t)
		is.NoError(err)
		is.Equal([]any{1, 2}, args)
	})

	t.Run("SetArguments", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1, 2)
		rm, _ := newReflectedMockCall(c)

		rm.SetArguments([]any{3, 4})

		args, err := rm.GetArguments()
		is := require.New(t)
		is.NoError(err)
		is.Equal([]any{3, 4}, args)
	})

	t.Run("CallReturnEmpty: nil err", func(t *testing.T) {
		c := (&mock.Mock{}).On("Test", 1, 2)
		rm, _ := newReflectedMockCall(c)
		rm.CallReturnEmpty(nil)
		// Because we don't have type information, a zero-value error is just ()
		require.Equal(t, mock.Arguments(mock.Arguments{}), c.ReturnArguments)
	})

	t.Run("CallReturnEmpty: with err", func(t *testing.T) {
		errTest := errors.New("ERROR")
		c := (&mock.Mock{}).On("Test", 1, 2)
		rm, _ := newReflectedMockCall(c)
		rm.CallReturnEmpty(errTest)
		require.Equal(t, mock.Arguments(mock.Arguments{errTest}), c.ReturnArguments)
	})
}

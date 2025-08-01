package tpp

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattavos/tpp/testdata"
)

// We unfortunately have to rely on reflection to get the type information for
// mock call arguments and returns. Of course, this makes us vulnerable to
// changes in testify/mockery breaking our code. These tests are a canary for
// breaking changes at this layer. If one of these tests fails, it's likely that
// something in testify/mockery has changed in an incompatible way.

var errTest = errors.New("TEST ERROR")

func TestReflect(t *testing.T) {
	_t := &testing.T{} // dummy testing.T for passing into code under test

	t.Run("newReflectedMockCall", func(t *testing.T) {
		c := testdata.NewMockIntyThing(_t).EXPECT().DoThing(1, 2)
		rm, err := newReflectedMockCall(c)
		require.NoError(t, err)
		require.NotNil(t, rm)
	})

	t.Run("GetArguments", func(t *testing.T) {
		c := testdata.NewMockIntyThing(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)
		args, err := rm.GetArguments()

		is := require.New(t)
		is.NoError(err)
		is.Equal([]any{1, 2}, args)
	})

	t.Run("SetArguments", func(t *testing.T) {
		c := testdata.NewMockIntyThing(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)

		rm.SetArguments([]any{3, 4})

		args, err := rm.GetArguments()
		is := require.New(t)
		is.NoError(err)
		is.Equal([]any{3, 4}, args)
	})

	t.Run("CallReturnEmpty: nil err", func(t *testing.T) {
		c := testdata.NewMockIntyThing(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)
		rm.CallReturnEmpty(nil)
		// DoThing returns (int, error), so empty is 0, nil
		require.Equal(t, mock.Arguments(mock.Arguments{0, nil}), c.ReturnArguments)
	})

	t.Run("CallReturnEmpty: with err", func(t *testing.T) {
		errTest := errors.New("ERROR")
		c := testdata.NewMockIntyThing(_t).EXPECT().DoThing(1, 2)
		rm, _ := newReflectedMockCall(c)
		rm.CallReturnEmpty(errTest)
		// DoThing returns (int, error)
		require.Equal(t, mock.Arguments(mock.Arguments{0, errTest}), c.ReturnArguments)
	})

	t.Run("CallReturn", func(t *testing.T) {
		// DoThing returns (int, error)
		type ret struct {
			rets   []any
			errVal error
		}

		for _, tt := range []struct {
			name       string
			withReturn ret
			wantPanic  bool
		}{
			{name: "OK: ret 42,nil", withReturn: ret{rets: []any{42, nil}}},
			{name: "OK: ret 0,nil", withReturn: ret{rets: []any{0, nil}}},
			{name: "OK: ret 1,err", withReturn: ret{rets: []any{1, errTest}}},
			{name: "OK: ret 0,err", withReturn: ret{rets: []any{0, errTest}}},
			{
				name:       "ERR: not enough returns",
				withReturn: ret{rets: []any{}},
				wantPanic:  true,
			},
			{
				name:       "ERR: too many returns",
				withReturn: ret{rets: []any{1, 2, 3}},
				wantPanic:  true,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				is := require.New(t)
				c := testdata.NewMockIntyThing(_t).EXPECT().DoThing(1, 2)
				rm, _ := newReflectedMockCall(c)

				call := func() {
					rm.CallReturn(tt.withReturn.rets, nil, false)
				}
				if tt.wantPanic {
					is.Panics(call)
				} else {
					call()
					is.Equal(mock.Arguments(tt.withReturn.rets), c.ReturnArguments)
				}
			})
		}
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

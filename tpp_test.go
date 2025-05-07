package tpp

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockImpl struct {
	mock.Mock
}

func (m *mockImpl) DoSomethingMockCall(x interface{}) *mock.Call {
	return m.On("DoSomething", x)
}

func (m *mockImpl) DoSomething(x int) bool {
	args := m.Called(x)
	return args.Bool(0)
}

func (m *mockImpl) DoSomethingVariadicMockCall(x interface{}, ys ...interface{}) *mock.Call {
	return m.On("DoSomethingVariadic", x, ys)
}

func (m *mockImpl) DoSomethingVariadic(x int, ys ...int) bool {
	args := m.Called(x, ys)
	return args.Bool(0)
}

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

		unexpected := Unexpected()
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

		unexpected := Unexpected()
		unexpected.Expectorise(fm)

		mockObj.AssertExpectations(t)
		is.Empty(mockObj.ExpectedCalls)
	})

	t.Run("Unsets a call", func(t *testing.T) {
		call := mockObj.On("DoSomething", 42).Return(true)

		unexpected := Unexpected()
		unexpected.Expectorise(call)

		mockObj.AssertExpectations(t)
		is.Empty(mockObj.ExpectedCalls)
	})
}

func TestExpects(t *testing.T) {
	is := require.New(t)

	t.Run("Expectorise works on non-variadic function", func(t *testing.T){
		mockObj := new(mockImpl)
		x := 3

		expects := OKs([]Call{
			{
				Given: []any{int(x)},
				Return: []any{false},
		}})

		expects.Expectorise(t, mockObj.DoSomethingMockCall, []any{false})

		res := mockObj.DoSomething(x)

		mockObj.AssertExpectations(t)
		is.False(res)
	})

	t.Run("Expectorise works on variadic function when no variadic args", func(t  *testing.T) {
		mockObj := new(mockImpl)
		x := 3
		
		expects := OKs([]Call{
			{
				Given: []any{int(x)},
				Return: []any{false},
		}})

		expects.Expectorise(t, mockObj.DoSomethingVariadicMockCall, []any{false})

		res := mockObj.DoSomethingVariadic(x)

		mockObj.AssertExpectations(t)
		is.False(res)
	})

	t.Run("Expectorise works on variadic function when 1 variadic arg", func(t  *testing.T) {
		mockObj := new(mockImpl)
		x := 3
		y := 1
		
		expects := OKs([]Call{
			{
				Given: []any{int(x), int(y)},
				Return: []any{false},
		}})

		expects.Expectorise(t, mockObj.DoSomethingVariadicMockCall, []any{false})

		res := mockObj.DoSomethingVariadic(x, y)

		mockObj.AssertExpectations(t)
		is.False(res)
	})

	t.Run("Expectorise works on variadic function when >1 variadic args", func(t  *testing.T) {
		mockObj := new(mockImpl)
		x := 3
		y1 := 1
		y2 := 2

		expects := OKs([]Call{
			{
				Given: []any{int(x), int(y1), int(y2)},
				Return: []any{false},
		}})

		expects.Expectorise(t, mockObj.DoSomethingVariadicMockCall, []any{false})

		res := mockObj.DoSomethingVariadic(x, y1, y2)

		mockObj.AssertExpectations(t)
		is.False(res)
	})

	t.Run("Expectorise works on variadic function when variadic args provided as array", func(t  *testing.T) {
		mockObj := new(mockImpl)
		x := 3
		ys := []int{1,2}

		expects := OKs([]Call{
			{
				Given: []any{int(x), ys},
				Return: []any{false},
		}})

		expects.Expectorise(t, mockObj.DoSomethingVariadicMockCall, []any{false})

		res := mockObj.DoSomethingVariadic(x, ys...)

		mockObj.AssertExpectations(t)
		is.False(res)
	})
}
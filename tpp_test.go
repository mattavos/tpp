package tpp

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockImpl struct {
	mock.Mock
}

func (m *mockImpl) DoSomething(x int) bool {
	args := m.Called(x)
	return args.Bool(0)
}

func TestUnexpectedWithArgumentMatcher(t *testing.T) {
	is := require.New(t)
	mockObj := new(mockImpl)

	// Create an argument matcher
	isEven := func(x int) bool {
		return x%2 == 0
	}
	argMatcher := mock.MatchedBy(isEven)
	call := mockObj.On("DoSomething", argMatcher).Return(true)

	unexpected := Unexpected()
	unexpected.Expectorise(call)

	mockObj.AssertExpectations(t)
	is.Empty(mockObj.ExpectedCalls)
}

func TestUnexpected(t *testing.T) {
	is := require.New(t)
	mockObj := new(mockImpl)

	call := mockObj.On("DoSomething", 42).Return(true)

	unexpected := Unexpected()
	unexpected.Expectorise(call)

	mockObj.AssertExpectations(t)
	is.Empty(mockObj.ExpectedCalls)
}

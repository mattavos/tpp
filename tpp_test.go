package tpp

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// FakerMock is a wrapper around a mock.Call, which resembles what we get from
// mockery.
type FakerMock struct {
	*mock.Call
}

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

func TestUnexpectedWithArgumentMatcherFakerMock(t *testing.T) {
	is := require.New(t)
	mockObj := new(mockImpl)

	// Create an argument matcher, and wrap it in a FakerMock to emulate mockery types
	isEven := func(x int) bool {
		return x%2 == 0
	}
	argMatcher := mock.MatchedBy(isEven)
	call := mockObj.On("DoSomething", argMatcher).Return(true)
	fm := FakerMock{call}

	unexpected := Unexpected()
	unexpected.Expectorise(fm)

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

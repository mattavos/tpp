package tpp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockInterface interface {
	DoSomething(x int) bool
}

type mockImpl struct {
	mock.Mock
}

func (m *mockImpl) DoSomething(x int) bool {
	args := m.Called(x)
	return args.Bool(0)
}

func TestUnexpectedWithArgumentMatcher(t *testing.T) {
	mockObj := new(mockImpl)

	// Create an argument matcher
	isEven := func(x int) bool {
		return x%2 == 0
	}
	matcher := mock.MatchedBy(isEven)

	// Set up the mock with the argument matcher
	call := mockObj.On("DoSomething", matcher).Return(true)

	// Create an unexpected expectation
	unexpected := Unexpected()

	// This should work now with our workaround
	unexpected.Expectorise(call)

	// Verify that the call was properly unset
	mockObj.AssertExpectations(t)
	assert.Empty(t, mockObj.ExpectedCalls)
}

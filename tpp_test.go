package tpp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

	// This should work, as we safely Unset.
	unexpected.Expectorise(call)

	// Verify that the call was properly unset
	mockObj.AssertExpectations(t)
	assert.Empty(t, mockObj.ExpectedCalls)
}

func TestUnexpected(t *testing.T) {
	mockObj := new(mockImpl)

	call := mockObj.On("DoSomething", 42).Return(true)

	// Create an unexpected expectation
	unexpected := Unexpected()

	// This should call Unset on the call
	unexpected.Expectorise(call)

	// Verify that the call was properly unset
	mockObj.AssertExpectations(t)
	assert.Empty(t, mockObj.ExpectedCalls)
}

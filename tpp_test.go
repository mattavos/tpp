package tpp

import (
	"sync"
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

func TestWithCallback(t *testing.T) {
	// Create a mock object
	mockObj := new(mockImpl)

	// Create an argument matcher
	isEven := func(x int) bool {
		return x%2 == 0
	}
	argMatcher := mock.MatchedBy(isEven)

	testCases := []struct {
		name            string
		expectSomething Expect
	}{
		{
			name:            "simple callback on a method that passes",
			expectSomething: OK(),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			call := mockObj.On("DoSomething", argMatcher)

			// Create a waitgroup
			var wg sync.WaitGroup
			wg.Add(1)

			// Set up the expectation with a callback
			tt.expectSomething.WithCallback(wg.Done).Expectorise(call)

			// Call the mock method
			mockObj.DoSomething(42)

			// Wait for the callback to be called
			wg.Wait()

			// Assert that the expectation was met
			mockObj.AssertExpectations(t)
		})
	}

}

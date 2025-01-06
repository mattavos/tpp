package tpp

import (
	"reflect"

	"github.com/stretchr/testify/mock"
)

// safeUnset safely unsets a mock call from its parent mock, handling argument matchers properly
func safeUnset(call *mock.Call) {
	// Get the parent mock through reflection
	parentValue := reflect.ValueOf(call).Elem().FieldByName("Parent")
	if !parentValue.IsValid() {
		return
	}
	parent := parentValue.Interface().(*mock.Mock)

	// Get the expected calls from the parent mock
	expectedCalls := reflect.ValueOf(parent).Elem().FieldByName("ExpectedCalls")
	if !expectedCalls.IsValid() {
		return
	}

	// Find and remove our call from the expected calls
	calls := expectedCalls.Interface().([]*mock.Call)
	for i, c := range calls {
		if c == call {
			// Remove this call from ExpectedCalls
			newCalls := append(calls[:i], calls[i+1:]...)
			expectedCalls.Set(reflect.ValueOf(newCalls))
			break
		}
	}
}

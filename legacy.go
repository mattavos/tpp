package tpp

import (
	"errors"

	"github.com/stretchr/testify/mock"
)

// Deprecated: the stuff in this file is deprecated.

// Deprecated: TrueOrNil is deprecated.
// TrueOrNil returns whether the given tri-state bool is true or nil.
func TrueOrNil(v *bool) bool {
	return v == nil || *v
}

// Deprecated: TestifyMock is deprecated.
// TestifyMock represents a testify mock. How meta.
type TestifyMock[T any, T2 any] interface {
	Return(T, error) T2
	Maybe() *mock.Call
	Unset() *mock.Call
}

// Deprecated: Expectorise is deprecated.
// Expectorise configures a TestifyMock to be used in configuration-driven tests.
//
// I.e., whether it should be expected to be called, a value to be called with,
// and whether it should return an error. This is to remove boilerplate code.
func Expectorise[T any, T2 any](m TestifyMock[T, T2], expect *bool, val T, err bool) {
	var expectoriseErr error
	if err {
		expectoriseErr = errors.New("TEST ERROR")
	}
	ExpectoriseWithErr(m, expect, val, expectoriseErr)
}

// Deprecated: ExpectoriseWithErr is deprecated.
// ExpectoriseWithErr is like Expectorise, but with a custom error.
func ExpectoriseWithErr[T any, T2 any](m TestifyMock[T, T2], expect *bool, val T, err error) {
	var empty T
	if expect == nil || *expect {
		if err != nil {
			m.Return(empty, err)
		} else {
			m.Return(val, nil)
		}
		if expect == nil {
			m.Maybe()
		}
	} else {
		m.Unset()
	}
}

// Deprecated: TestifyMock2 is deprecated.
// TestifyMock2 represents a testify mock. How meta.
type TestifyMock2[R any, R2 any, T any] interface {
	Return(R, R2, error) T
	Maybe() *mock.Call
	Unset() *mock.Call
}

// Deprecated: Expectorise2 is deprecated.
// Expectorise2 configures a TestifyMock to be used in configuration-driven tests.
//
// Takes two return values as opposed to Expectorise which takes one return value.
//
// I.e., whether it should be expected to be called, a value to be called with,
// and whether it should return an error. This is to remove boilerplate code.
func Expectorise2[R any, R2 any, T any](
	m TestifyMock2[R, R2, T],
	expect *bool,
	ret R,
	ret2 R2,
	err bool,
) {
	var empty R
	var empty2 R2
	if expect == nil || *expect {
		if err {
			m.Return(empty, empty2, errors.New("TEST ERROR"))
		} else {
			m.Return(ret, ret2, nil)
		}
		if expect == nil {
			m.Maybe()
		}
	} else {
		m.Unset()
	}
}

// Deprecated: TestifyMock0 is deprecated.
// TestifyMock0 represents a testify mock. How meta.
type TestifyMock0[T any] interface {
	Return(error) T
	Maybe() *mock.Call
	Unset() *mock.Call
}

// Deprecated: Expectorise0 is deprecated.
// Expectorise0 configures a TestifyMock to be used in configuration-driven tests.
//
// Takes one return value as opposed to Expectorise which takes one return value.
//
// I.e., whether it should be expected to be called, a value to be called with,
// and whether it should return an error. This is to remove boilerplate code.
func Expectorise0[T any](
	m TestifyMock0[T],
	expect *bool,
	err bool,
) {
	if expect == nil || *expect {
		if err {
			m.Return(errors.New("TEST ERROR"))
		} else {
			m.Return(nil)
		}
		if expect == nil {
			m.Maybe()
		}
	} else {
		m.Unset()
	}
}

// TestifyMockNoErr is like TestifyMock, but with no error in Return().
type TestifyMockNoErr[T any, T2 any] interface {
	Return(T) T2
	Maybe() *mock.Call
	Unset() *mock.Call
}

// ExpectoriseNoErr is like Expectorise, but without a returned error.
func ExpectoriseNoErr[T any, T2 any](m TestifyMockNoErr[T, T2], expect *bool, val T) {
	if expect == nil || *expect {
		m.Return(val)
		if expect == nil {
			m.Maybe()
		}
	} else {
		m.Unset()
	}
}

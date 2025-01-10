# T++: A Meta-Test Framework For Go

[![Go](https://github.com/mattavos/tpp/actions/workflows/tpp.yml/badge.svg)](https://github.com/mattavos/tpp/actions/workflows/tpp.yml)

>[!CAUTION]
> This is a work-in-progress project. The implementation is rough and the interface is liable to change. Please use with care and conservatism.

![Logo](/doc/tpp-logo.png)


## What is T++?

This package has some generic helpers to facilitate writing configuration driven tests.
These are where one single meta-test is written which is then configured by a passed in struct, which defines the actual test behaviour.

## Why T++?
Using this package, you can write a single test function that can be configured to run multiple test cases. This is useful when you have a lot of similar test cases that only differ in the input data or the expected output.

It's also clearer to see what each test is doing, how they differ, and reason about cases that could be missing.
Fewer lines of code are needed to write the tests, and the tests are more readable and maintainable as a result.

## How to use T++?

### Installation

Once you have installed the package, you can start using it in your tests.

```bash
go get github.com/mattavos/tpp
```

### Example

<details>
<summary>Click to expand</summary>

```go
package example_test

import (
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/your/repo/mocks"
    "github.com/your/repo/subject"
    "github.com/mattavos/tpp"
)

func TestXXX(t *testing.T) {
	for _, tt := range []struct {
		name    string
		getFoo  tpp.Expect
		wantErr bool
	}{
		{
			name:    "OK",
			getFoo:  tpp.OK("foo"),
			wantErr: false
		},
		{
			name:    "ERR: getFoo",
			getFoo:  tpp.Err(),
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mock := mymocks.NewBar(t)
			tt.getFoo.Expectorise(mock.EXPECT().GetFoo())

			subject := subject.New(mock)
			err := subject.XXX()

			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}
```
</details>

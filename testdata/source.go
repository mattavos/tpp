package testdata

import "context"

// The generated mockery mocks in this package are from the following definitions:

type IntyThing interface {
	DoThing(a, b int) (int, error)
}

type Struct struct {
	A int
	B int
}

type StructyThing interface {
	DoThing(context.Context, *Struct) (*Struct, error)
}

type EmptyThing interface {
	DoThing()
}

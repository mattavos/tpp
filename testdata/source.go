package testdata

import "context"

// The generated mockery mocks in this package are from the following definitions:

type EmptyThing interface {
	DoThing()
}

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

type ConcreteStructyThing interface {
	DoThing(context.Context, Struct) (Struct, error)
}

type SliceyThing interface {
	DoThing(context.Context, []int) ([]int, error)
}

type FuncyThing interface {
	DoThing(func(int) int) func(int) int
}

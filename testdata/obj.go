package obj

type Obj interface {
	DoThing(a, b int) (int, error)
}

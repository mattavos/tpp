package tpp

// We unfortunately have to rely on reflection to get the type information for
// mock call arguments and returns. Of course, this makes us vulnerable to
// changes in testify/mockery breaking our code. These tests are a canary for
// breaking changes at this layer. If one of these tests fails, it's likely that
// something in testify/mockery has changed in an incompatible way.

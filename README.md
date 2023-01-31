# errortree
[![Go Reference](https://pkg.go.dev/badge/github.com/convto/errortree.svg)](https://pkg.go.dev/github.com/convto/errortree) [![Go Report Card](https://goreportcard.com/badge/github.com/convto/bit)](https://goreportcard.com/report/github.com/convto/errortree) [![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Package errortree provides multiple-error matching considering the tree structure of errors in Go1.20 and later.

This package also provides the ability to retrieve errors from the tree, using generics to keep the type info from the caller, and returns the search result with an arbitrary concrete type.

## Why created the std errors extension?

Standard errors will support tree structures starting with Go1.20.
Its design is as follows:

- Depth-first tree traversal
- If the target matches, the traversal ends and the result is returned
- **Not all branches in the tree have to match**

So it is difficult to satisfying some of the requirements!

**Requirement A: Check all branches in the tree whether err and target are equal**

For example, sometimes users want to ignore err or only logging for target errors, like:

```go
switch err := ExecuteSomeFunc(); {
case errors.Is(err, TargetErr):
    // ignore or logging...
case err != nil:
    return err
}
```

This code checks for errors that can be ignored.
In this case, if `multiErr{targetErr, DifferentErr}` or similar is passed as err, `errors.Is(err, target)` will return true even though it contains `DifferentErr`.

In some contexts, there will be use cases where user want to make sure that all branches match more strictly to report whether the err tree is equal to target. std `error.Is()` cannot be used for such a purpose.

So, created following functionlity for a more exact comparison.

```go
ExactlyIs(err error, target error) bool
```

**Requirement B: Extract all matching targets in the tree**

The std `errors.As()` retrieves only the first error that matches the target, but there are some cases where the user may want to retrieve all matching errors for handling purposes.

For this reason, this package provides the following functionality to retrieve errors of a concrete type using generics.

```go
Scan[T any](err error, targetT) []T
```

`Scan()` could be done a little better, function form is a little bit less smart.

## Example

ExactlyIs

```go
erra := errors.New("error A")
err := multiErr{
	wrapErr{
		multiErr{
			erra,
			erra,
		},
	},
	wrapErr{erra},
	multiErr{
		erra,
		erra,
		errors.New("wrong error"),
	},
}
result := errortree.ExactlyIs(err, erra)
fmt.Println(result)

// Output:
// false
```

Scan

```go
err := multiErr{
	multiErr{
		errors.New("error"),
		&fs.PathError{Op: "poser A"},
	},
	wrapErr{&fs.PathError{Op: "poser B"}},
	multiErr{
		errors.New("error"),
		errors.New("error"),
		wrapErr{&fs.PathError{Op: "poser C"}},
	},
}
var p *fs.PathError
matched := errortree.Scan(err, p)
for _, v := range matched {
	fmt.Printf("%s\n", v.Op)
}

// Output:
// poser A
// poser B
// poser C
```

# errortree
Package errortree provides multiple-error matching considering the tree structure of errors in Go1.20 and later.

This package uses generics to keep the type info from the caller, and returns the search result with an arbitrary concrete type.

## Useage

Can extract all matching errors with their concrete types.

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

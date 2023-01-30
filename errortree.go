// Package errortree provides multiple-error matching considering the tree structure
// of errors in Go1.20 and later.
//
// This package uses generics to keep the type info from the caller, and returns the
// search result with an arbitrary concrete type.
package errortree

import (
	"reflect"
)

// ExactlyIs reports whether all branches in the err's tree matches the target.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling Unwrap. When err wraps multiple errors, Is examines err followed by a
// depth-first traversal of its children.
//
// For example, ExactlyIs(err, target) will return true on the following tree
// because all branches match target when viewed from the root.
//
//	err
//	├── target
//	├── wrapErr
//	│   └── multiErr
//	│       ├── target
//	│       └── target
//	└── multiErr
//	    ├── wrapErr
//	    │   └── target
//	    └── target
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func ExactlyIs(err error, target error) bool {
	if target == nil {
		return err == target
	}

	isComparable := reflect.TypeOf(target).Comparable()
	for {
		if isComparable && err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			errs := x.Unwrap()
			if len(errs) == 0 {
				return false
			}
			for _, err := range errs {
				if !ExactlyIs(err, target) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
}

// Scan finds all matches to target in err's tree, always traverses all trees even
// if target is found during the search, if no matches are found, it returns nil.
//
// For example, execute Scan(err, target) on the following tree
//
//	err
//	├── targetA(assignable to `target`)
//	├── wrapErr
//	│   └── multiErr
//	│       ├── targetB(assignable to `target`)
//	│       └── targetC(assignable to `target`)
//	└── multiErr
//	    ├── wrapErr
//	    │   └── targetD(assignable to `target`)
//	    └── targetE(assignable to `target`)
//
// It returns `[]target{targetA, targetB, targetC, targetD, targetE}`.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling Unwrap. When err wraps multiple errors, Scan examines err followed by a
// depth-first traversal of its children.
//
// An error matches target if the err's concrete value is assignable to the value
// pointed to by target.
//
// Scan panics if target is not implements error, or to any interface type.
//
// Note target parameter accepts an interface, so setting `interface{}` or `any` to
// target will match all nodes!
func Scan[T any](err error, target T) (matched []T) {
	if err == nil {
		return nil
	}
	targetType := reflect.TypeOf(target)
	if targetType == nil {
		targetType = reflect.TypeOf(&target).Elem()
	}
	if targetType.Kind() != reflect.Interface && !targetType.Implements(errorType) {
		panic("errortree: target must be interface or implement error")
	}
	for {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			if v, ok := err.(T); ok {
				matched = append(matched, v)
			}
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return matched
			}
		case interface{ Unwrap() []error }:
			for _, v := range x.Unwrap() {
				m := Scan(v, target)
				if len(m) > 0 {
					matched = append(matched, m...)
				}
			}
			return matched
		default:
			return matched
		}
	}
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

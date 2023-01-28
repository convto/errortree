package errortree

import (
	"reflect"
)

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

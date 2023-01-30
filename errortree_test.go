package errortree_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/convto/errortree"
)

func TestExactlyIs(t *testing.T) {
	err1 := errors.New("1")
	erra := wrapErr{err1}
	errb := wrapErr{erra}

	err3 := errors.New("3")

	poser := &poser{"either 1 or 3", func(err error) bool {
		return err == err1 || err == err3
	}}

	testCases := []struct {
		err    error
		target error
		match  bool
	}{
		{nil, nil, true},
		{err1, nil, false},
		{err1, err1, true},
		{erra, err1, true},
		{errb, err1, true},
		{err1, err3, false},
		{erra, err3, false},
		{errb, err3, false},
		{poser, err1, true},
		{poser, err3, true},
		{poser, erra, false},
		{poser, errb, false},
		{errorUncomparable{}, errorUncomparable{}, true},
		{errorUncomparable{}, &errorUncomparable{}, false},
		{&errorUncomparable{}, errorUncomparable{}, true},
		{&errorUncomparable{}, &errorUncomparable{}, false},
		{errorUncomparable{}, err1, false},
		{&errorUncomparable{}, err1, false},
		{multiErr{}, err1, false},
		{multiErr{err1, err3}, err1, false},
		{multiErr{err1, err3}, errors.New("x"), false},
		{multiErr{err3, errb}, errb, false},
		{multiErr{errb, errb}, errb, true},
		{multiErr{errb, errb}, erra, true},
		{multiErr{poser}, err1, true},
		{multiErr{poser}, err3, true},
		{multiErr{nil}, nil, false},
		{multiErr{errb, erra, multiErr{errb, erra}}, err1, true},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if got := errortree.ExactlyIs(tc.err, tc.target); got != tc.match {
				t.Errorf("Is(%v, %v) = %v, want %v", tc.err, tc.target, got, tc.match)
			}
		})
	}
}

type poser struct {
	msg string
	f   func(error) bool
}

var poserPathErr = &fs.PathError{Op: "poser"}

func (p *poser) Error() string     { return p.msg }
func (p *poser) Is(err error) bool { return p.f(err) }

func TestScan(t *testing.T) {
	var errT errorT
	var errP *fs.PathError
	var timeout interface{ Timeout() bool }
	var p *poser
	_, errF := os.Open("non-existing")
	poserErr := &poser{"oh no", nil}

	testCases := []struct {
		err    error
		target any
		want   []error
	}{
		{
			nil,
			errP,
			[]error{errP},
		},
		{
			wrapErr{errorT{"T"}},
			errT,
			[]error{errorT{"T"}},
		},
		{
			errF,
			errP,
			[]error{errF},
		},
		{
			errorT{},
			errP,
			nil,
		},
		{
			wrapErr{nil},
			errT,
			nil,
		},
		{
			&poser{"error", nil},
			errT,
			[]error{errorT{"poser"}},
		},
		{
			&poser{"path", nil},
			errP,
			[]error{poserPathErr},
		},
		{
			poserErr,
			p,
			[]error{poserErr},
		},
		{
			errors.New("err"),
			timeout,
			nil,
		},
		{
			errF,
			timeout,
			[]error{errF},
		},
		{
			wrapErr{errF},
			timeout,
			[]error{errF},
		},
		{
			multiErr{},
			errT,
			nil,
		},
		{
			multiErr{errors.New("a"), errorT{"T"}},
			errT,
			[]error{errorT{"T"}},
		},
		{
			multiErr{errorT{"T"}, errors.New("a")},
			errT,
			[]error{errorT{"T"}},
		},
		{
			multiErr{errorT{"a"}, errorT{"b"}},
			errT,
			[]error{errorT{"a"}, errorT{"b"}},
		},
		{
			multiErr{multiErr{errors.New("a"), errorT{"a"}}, errorT{"b"}},
			errT,
			[]error{errorT{"a"}, errorT{"b"}},
		},
		{
			multiErr{wrapErr{errF}},
			timeout,
			[]error{errF},
		},
		{
			multiErr{nil},
			&errT,
			nil,
		},
	}
	for i, tc := range testCases {
		name := fmt.Sprintf("%d:Scan(Errorf(..., %v), %v)", i, tc.err, tc.target)
		t.Run(name, func(t *testing.T) {
			matched := errortree.Scan(tc.err, tc.target)
			if reflect.DeepEqual(matched, tc.want) {
				t.Fatalf("match: got %#v; want %#v", matched, tc.want)
			}
		})
	}
}

func TestScanValidation(t *testing.T) {
	var s string
	testCases := []any{
		(*int)(nil),
		"error",
		&s,
	}
	err := errors.New("error")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%T(%v)", tc, tc), func(t *testing.T) {
			defer func() {
				recover()
			}()
			if matched := errortree.Scan(err, tc); len(matched) != 0 {
				t.Errorf("Scan(err, %T(%v)) result present, want empty", tc, tc)
				return
			}
		})
	}
}

type errorT struct{ s string }

func (e errorT) Error() string { return fmt.Sprintf("errorT(%s)", e.s) }

type errorUncomparable struct {
	f []string
}

func (errorUncomparable) Error() string {
	return "uncomparable error"
}

func (errorUncomparable) Is(target error) bool {
	_, ok := target.(errorUncomparable)
	return ok
}

type wrapErr struct {
	err error
}

func (e wrapErr) Error() string { return "wrapErr" }
func (e wrapErr) Unwrap() error { return e.err }

type multiErr []error

func (m multiErr) Error() string   { return "multiError" }
func (m multiErr) Unwrap() []error { return []error(m) }

func ExampleExactlyIs() {
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
}

func ExampleScan() {
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
}

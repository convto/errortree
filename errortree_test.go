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

type poser struct {
	msg string
	f   func(error) bool
}

var poserPathErr = &fs.PathError{Op: "poser"}

func (p *poser) Error() string { return p.msg }

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

type wrapErr struct {
	err error
}

func (e wrapErr) Error() string { return "wrapErr" }
func (e wrapErr) Unwrap() error { return e.err }

type multiErr []error

func (m multiErr) Error() string   { return "multiError" }
func (m multiErr) Unwrap() []error { return []error(m) }

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

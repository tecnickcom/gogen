package errutil

import (
	"errors"
	"testing"
)

var errBench = errors.New("bench error")

func BenchmarkTrace(b *testing.B) {
	var err error

	for range b.N {
		err = Trace(errBench)
	}

	_ = err
}

func BenchmarkJoinFnError(b *testing.B) {
	fn := func() error {
		return errBench
	}

	for range b.N {
		var err error

		JoinFnError(&err, fn)
	}
}

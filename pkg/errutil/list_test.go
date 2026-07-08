package errutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorsNil(t *testing.T) {
	t.Parallel()

	require.Nil(t, Errors(nil))
}

func TestErrorsSingle(t *testing.T) {
	t.Parallel()

	err := errors.New("single error")

	errs := Errors(err)

	require.Len(t, errs, 1)
	require.Same(t, err, errs[0])
}

func TestErrorsJoined(t *testing.T) {
	t.Parallel()

	first := errors.New("first error")
	second := errors.New("second error")

	errs := Errors(errors.Join(first, second))

	require.Len(t, errs, 2)
	require.ErrorIs(t, errs[0], first)
	require.ErrorIs(t, errs[1], second)
}

func TestErrorsJoinedReturnsCopy(t *testing.T) {
	t.Parallel()

	first := errors.New("first error")
	second := errors.New("second error")
	joined := errors.Join(first, second)

	errs := Errors(joined)
	errs[0] = nil // mutating the result must not corrupt the aggregate

	require.ErrorIs(t, joined, first)
	require.ErrorIs(t, joined, second)
}

package errutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoinFnError(t *testing.T) {
	t.Parallel()

	var err error

	JoinFnError(&err, func() error {
		return errors.New("first error")
	})

	require.EqualError(t, err, "first error")

	JoinFnError(&err, func() error {
		return errors.New("second error")
	})

	require.EqualError(t, err, "first error\nsecond error")

	JoinFnError(&err, func() error {
		return nil
	})

	require.EqualError(t, err, "first error\nsecond error")

	var nilErr error

	JoinFnError(&nilErr, func() error {
		return nil
	})

	require.NoError(t, nilErr)
}

func TestJoinFnErrorNilFunc(t *testing.T) {
	t.Parallel()

	var err error

	JoinFnError(&err, nil)

	require.ErrorIs(t, err, ErrNilErrorFunc)

	primary := errors.New("primary error")
	err = primary

	JoinFnError(&err, nil)

	require.ErrorIs(t, err, primary)
	require.ErrorIs(t, err, ErrNilErrorFunc)
}

func TestJoinFnErrorNilPointer(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		JoinFnError(nil, func() error {
			return errors.New("dropped error")
		})
	})
}

func TestJoinFnErrorPreservesPrimaryOnNilResult(t *testing.T) {
	t.Parallel()

	primary := errors.New("primary error")
	err := primary

	JoinFnError(&err, func() error {
		return nil
	})

	// The primary error must be preserved unchanged when the cleanup function
	// reports no error: its concrete type must not be re-wrapped in a joinError.
	require.Equal(t, fmt.Sprintf("%T", primary), fmt.Sprintf("%T", err))
	require.EqualError(t, err, "primary error")
}

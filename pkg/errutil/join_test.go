package errutil

import (
	"errors"
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

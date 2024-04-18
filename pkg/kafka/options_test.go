package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_WithSessionTimeout(t *testing.T) {
	t.Parallel()

	v := time.Second * 17

	cfg := &config{}
	WithSessionTimeout(v)(cfg)
	require.Equal(t, v, cfg.sessionTimeout)
}

func Test_WithFirstOffset(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	WithFirstOffset()(cfg)
	require.Equal(t, int64(-2), cfg.startOffset)
}

func Test_WithMessageEncodeFunc(t *testing.T) {
	t.Parallel()

	ret := []byte("test_data_001")
	f := func(_ context.Context, _ any) ([]byte, error) {
		return ret, nil
	}

	conf := &config{}
	WithMessageEncodeFunc(f)(conf)

	d, err := conf.messageEncodeFunc(context.TODO(), "")
	require.NoError(t, err)
	require.Equal(t, ret, d)
}

func Test_WithMessageDecodeFunc(t *testing.T) {
	t.Parallel()

	f := func(_ context.Context, _ []byte, _ any) error {
		return nil
	}

	conf := &config{}
	WithMessageDecodeFunc(f)(conf)
	require.NoError(t, conf.messageDecodeFunc(context.TODO(), nil, ""))
}

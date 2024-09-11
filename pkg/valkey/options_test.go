package valkey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func Test_WithMessageEncodeFunc(t *testing.T) {
	t.Parallel()

	ret := "test_data_001"
	f := func(_ context.Context, _ any) (string, error) {
		return ret, nil
	}

	conf := &cfg{}
	WithMessageEncodeFunc(f)(conf)

	d, err := conf.messageEncodeFunc(context.TODO(), "")
	require.NoError(t, err)
	require.Equal(t, ret, d)
}

func Test_WithMessageDecodeFunc(t *testing.T) {
	t.Parallel()

	f := func(_ context.Context, _ string, _ any) error {
		return nil
	}

	conf := &cfg{}
	WithMessageDecodeFunc(f)(conf)
	require.NoError(t, conf.messageDecodeFunc(context.TODO(), "", ""))
}

func Test_WithChannels(t *testing.T) {
	t.Parallel()

	chns := []string{"alpha", "beta", "gamma"}

	conf := &cfg{}
	WithChannels(chns...)(conf)
	require.Len(t, conf.channels, 3)
}

func Test_WithValkeyClient(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewClient(ctrl)

	conf := &cfg{}
	WithValkeyClient(client)(conf)
	require.Equal(t, client, conf.vkclient)
}

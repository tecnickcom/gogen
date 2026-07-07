package timeutil

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dur  Duration
		want string
	}{
		{
			name: "seconds",
			dur:  Duration(13 * time.Second),
			want: "13s",
		},
		{
			name: "minutes",
			dur:  Duration(17 * time.Minute),
			want: "17m0s",
		},
		{
			name: "hours",
			dur:  Duration(7*time.Hour + 11*time.Minute + 13*time.Second),
			want: "7h11m13s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.dur.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, "\""+tt.want+"\"", string(got))

			got, err = json.Marshal(tt.dur)
			require.NoError(t, err)
			require.Equal(t, "\""+tt.want+"\"", string(got))
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		want    Duration
		wantErr bool
	}{
		{
			name:    "empty",
			data:    []byte(``),
			wantErr: true,
		},
		{
			name:    "empty string",
			data:    []byte(`""`),
			wantErr: true,
		},
		{
			name:    "invalid string",
			data:    []byte(`"-"`),
			wantErr: true,
		},
		{
			name:    "invalid type",
			data:    []byte(`{"a":"b"}`),
			wantErr: true,
		},
		{
			name:    "numeric value overflowing float64",
			data:    []byte(`1e400`),
			wantErr: true,
		},
		{
			name:    "integer overflowing int64",
			data:    []byte(`10000000000000000000`), // MaxInt64 + ~0.8e18
			wantErr: true,
		},
		{
			name:    "exponent overflowing int64",
			data:    []byte(`1e19`),
			wantErr: true,
		},
		{
			name:    "exponent underflowing int64",
			data:    []byte(`-1e19`),
			wantErr: true,
		},
		{
			name: "seconds",
			data: []byte(`"13s"`),
			want: Duration(13 * time.Second),
		},
		{
			name: "minutes",
			data: []byte(`"17m0s"`),
			want: Duration(17 * time.Minute),
		},
		{
			name: "hours",
			data: []byte(`"73h0m0s"`),
			want: Duration(73 * time.Hour),
		},
		{
			name: "number",
			data: []byte(`123456789`),
			want: Duration(123456789),
		},
		{
			name: "zero number",
			data: []byte(`0`),
			want: Duration(0),
		},
		{
			name: "negative number",
			data: []byte(`-17`),
			want: Duration(-17),
		},
		{
			name: "large integer number beyond float64 precision",
			data: []byte(`9007199254740993`), // 2^53 + 1
			want: Duration(9007199254740993),
		},
		{
			name: "max int64 number",
			data: []byte(`9223372036854775807`),
			want: Duration(math.MaxInt64),
		},
		{
			name: "min int64 number",
			data: []byte(`-9223372036854775808`),
			want: Duration(math.MinInt64),
		},
		{
			name: "fractional number",
			data: []byte(`1.5`),
			want: Duration(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dur Duration

			err := dur.UnmarshalJSON(tt.data)
			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, int64(tt.want), int64(dur))

			var d Duration

			err = json.Unmarshal(tt.data, &d)
			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, int64(tt.want), int64(d))
		})
	}
}

func TestDuration_MarshalText(t *testing.T) {
	t.Parallel()

	got, err := Duration(7*time.Hour + 11*time.Minute + 13*time.Second).MarshalText()
	require.NoError(t, err)
	require.Equal(t, "7h11m13s", string(got))
}

func TestDuration_UnmarshalText(t *testing.T) {
	t.Parallel()

	var d Duration

	err := d.UnmarshalText([]byte("1h30m"))
	require.NoError(t, err)
	require.Equal(t, int64(90*time.Minute), int64(d))

	err = d.UnmarshalText([]byte("nope"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidDuration)
}

func TestDuration_JSONMapKey(t *testing.T) {
	t.Parallel()

	m := map[Duration]int{
		Duration(90 * time.Minute): 3,
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)
	require.JSONEq(t, `{"1h30m0s":3}`, string(data))

	var got map[Duration]int

	err = json.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, 3, got[Duration(90*time.Minute)])
}

func TestDuration_UnmarshalJSON_Null(t *testing.T) {
	t.Parallel()

	dur := Duration(13 * time.Second)

	err := dur.UnmarshalJSON([]byte(`null`))
	require.NoError(t, err)
	require.Equal(t, int64(13*time.Second), int64(dur), "null must be a no-op and leave the value unchanged")

	err = json.Unmarshal([]byte(`null`), &dur)
	require.NoError(t, err)
	require.Equal(t, int64(13*time.Second), int64(dur), "null must be a no-op and leave the value unchanged")
}

func TestDuration_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dur  Duration
	}{
		{
			name: "large integer beyond float64 precision",
			dur:  Duration(9007199254740993), // 2^53 + 1
		},
		{
			name: "max int64",
			dur:  Duration(math.MaxInt64),
		},
		{
			name: "small duration",
			dur:  Duration(13 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.dur)
			require.NoError(t, err)

			var got Duration

			err = json.Unmarshal(data, &got)
			require.NoError(t, err)
			require.Equal(t, int64(tt.dur), int64(got), "duration did not round-trip exactly")
		})
	}
}

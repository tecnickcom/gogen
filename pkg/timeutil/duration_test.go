package timeutil

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dur  Duration
		want []byte
	}{
		{
			name: "seconds",
			dur:  Duration(13 * time.Second),
			want: []byte(`"13s"`),
		},
		{
			name: "minutes",
			dur:  Duration(17 * time.Minute),
			want: []byte(`"17m0s"`),
		},
		{
			name: "hours",
			dur:  Duration(7*time.Hour + 11*time.Minute + 13*time.Second),
			want: []byte(`"7h11m13s"`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.dur.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_json_Marshal(t *testing.T) {
	t.Parallel()

	type testData struct {
		Time Duration `json:"Time"`
	}

	tests := []struct {
		name string
		data testData
		want []byte
	}{
		{
			name: "seconds",
			data: testData{Time: Duration(13 * time.Second)},
			want: []byte(`{"Time":"13s"}`),
		},
		{
			name: "minutes",
			data: testData{Time: Duration(17 * time.Minute)},
			want: []byte(`{"Time":"17m0s"}`),
		},
		{
			name: "hours",
			data: testData{Time: Duration(7*time.Hour + 11*time.Minute + 13*time.Second)},
			want: []byte(`{"Time":"7h11m13s"}`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.data)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dur Duration

			err := dur.UnmarshalJSON(tt.data)
			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, int64(tt.want), int64(dur))
		})
	}
}

func Test_json_Unmarshal(t *testing.T) {
	t.Parallel()

	type testData struct {
		Time Duration `json:"Time"`
	}

	tests := []struct {
		name    string
		data    []byte
		want    Duration
		wantErr bool
	}{
		{
			name: "seconds",
			data: []byte(`{"Time":"13s"}`),
			want: Duration(13 * time.Second),
		},
		{
			name: "minutes",
			data: []byte(`{"Time":"17m0s"}`),
			want: Duration(17 * time.Minute),
		},
		{
			name: "hours",
			data: []byte(`{"Time":"7h11m13s"}`),
			want: Duration(7*time.Hour + 11*time.Minute + 13*time.Second),
		},
		{
			name: "number",
			data: []byte(`{"Time":123456789}`),
			want: Duration(123456789),
		},
		{
			name: "zero number",
			data: []byte(`{"Time":0}`),
			want: Duration(0),
		},
		{
			name: "negative number",
			data: []byte(`{"Time":-17}`),
			want: Duration(-17),
		},
		{
			name:    "null",
			data:    []byte(`{"Time":null}`),
			wantErr: true,
		},
		{
			name:    "empty string",
			data:    []byte(`{"Time":""}`),
			wantErr: true,
		},
		{
			name:    "invalid string",
			data:    []byte(`{"Time":"-"}`),
			wantErr: true,
		},
		{
			name:    "invalid type",
			data:    []byte(`{"Time":{"a":"b"}}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d testData

			err := json.Unmarshal(tt.data, &d)
			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, int64(tt.want), int64(d.Time))
		})
	}
}

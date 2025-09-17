package timeutil

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDateTime_MarshalJSON(t *testing.T) {
	t.Parallel()

	reftime := time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", -7*60*60))

	tests := []struct {
		name string
		dt   any
		want string
	}{
		{
			name: "TRFC3339 zero value",
			dt:   new(DateTime[TRFC3339]),
			want: "0001-01-01T00:00:00Z", // time.RFC3339 zero value
		},
		{
			name: "TJira zero value",
			dt:   new(DateTime[TJira]),
			want: "0001-01-01T00:00:00.000+0000", // time.Jira zero value
		},
		{
			name: "TLayout",
			dt:   DateTime[TLayout](reftime),
			want: "01/02 03:04:05PM '06 -0700", // time.Layout
		},
		{
			name: "TANSIC",
			dt:   DateTime[TANSIC](reftime),
			want: "Mon Jan  2 15:04:05 2006", // time.ANSIC
		},
		{
			name: "TUnixDate",
			dt:   DateTime[TUnixDate](reftime),
			want: "Mon Jan  2 15:04:05 MST 2006", // time.UnixDate
		},
		{
			name: "TRubyDate",
			dt:   DateTime[TRubyDate](reftime),
			want: "Mon Jan 02 15:04:05 -0700 2006", // time.RubyDate
		},
		{
			name: "TRFC822",
			dt:   DateTime[TRFC822](reftime),
			want: "02 Jan 06 15:04 MST", // time.RFC822
		},
		{
			name: "TRFC822Z",
			dt:   DateTime[TRFC822Z](reftime),
			want: "02 Jan 06 15:04 -0700", // time.RFC822Z
		},
		{
			name: "TRFC850",
			dt:   DateTime[TRFC850](reftime),
			want: "Monday, 02-Jan-06 15:04:05 MST", // time.RFC850
		},
		{
			name: "TRFC1123",
			dt:   DateTime[TRFC1123](reftime),
			want: "Mon, 02 Jan 2006 15:04:05 MST", // time.RFC1123
		},
		{
			name: "TRFC1123Z",
			dt:   DateTime[TRFC1123Z](reftime),
			want: "Mon, 02 Jan 2006 15:04:05 -0700", // time.RFC1123Z
		},
		{
			name: "TRFC3339",
			dt:   DateTime[TRFC3339](reftime),
			want: "2006-01-02T15:04:05-07:00", // time.RFC3339
		},
		{
			name: "TRFC3339Nano",
			dt:   DateTime[TRFC3339Nano](reftime),
			want: "2006-01-02T15:04:05-07:00", // time.RFC3339Nano
		},
		{
			name: "TKitchen",
			dt:   DateTime[TKitchen](reftime),
			want: "3:04PM", // time.Kitchen
		},
		{
			name: "TStamp",
			dt:   DateTime[TStamp](reftime),
			want: "Jan  2 15:04:05", // time.Stamp
		},
		{
			name: "TStampMilli",
			dt:   DateTime[TStampMilli](reftime),
			want: "Jan  2 15:04:05.000", // time.StampMilli
		},
		{
			name: "TStampMicro",
			dt:   DateTime[TStampMicro](reftime),
			want: "Jan  2 15:04:05.000000", // time.StampMicro
		},
		{
			name: "TStampNano",
			dt:   DateTime[TStampNano](reftime),
			want: "Jan  2 15:04:05.000000000", // time.StampNano
		},
		{
			name: "TDateTime",
			dt:   DateTime[TDateTime](reftime),
			want: "2006-01-02 15:04:05", // time.DateTime
		},
		{
			name: "TDateOnly",
			dt:   DateTime[TDateOnly](reftime),
			want: "2006-01-02", // time.DateOnly
		},
		{
			name: "TTimeOnly",
			dt:   DateTime[TTimeOnly](reftime),
			want: "15:04:05", // time.TimeOnly
		},
		{
			name: "TJira",
			dt:   DateTime[TJira](reftime),
			want: TimeJiraFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			marshaler, ok := tt.dt.(json.Marshaler)
			require.True(t, ok, "dt does not implement json.Marshaler")

			got, err := marshaler.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, "\""+tt.want+"\"", string(got))

			got, err = json.Marshal(tt.dt)
			require.NoError(t, err)
			require.Equal(t, "\""+tt.want+"\"", string(got))
		})
	}
}

func TestDateTime_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		tobj    any
		want    time.Time
		wantErr bool
	}{
		{
			name:    "nil",
			tobj:    new(DateTime[TRFC3339]),
			wantErr: true,
		},
		{
			name:    "empty",
			tobj:    new(DateTime[TRFC3339]),
			data:    []byte(``),
			wantErr: true,
		},
		{
			name:    "empty string",
			tobj:    new(DateTime[TRFC3339]),
			data:    []byte(`""`),
			wantErr: true,
		},
		{
			name:    "invalid string",
			tobj:    new(DateTime[TRFC3339]),
			data:    []byte(`"-"`),
			wantErr: true,
		},
		{
			name:    "invalid type",
			tobj:    new(DateTime[TRFC3339]),
			data:    []byte(`{"a":"b"}`),
			wantErr: true,
		},
		{
			name: "TRFC3339 zero value",
			tobj: new(DateTime[TRFC3339]),
			data: []byte(`"0001-01-01T00:00:00Z"`),
			want: time.Time{},
		},
		{
			name: "TJira zero value",
			tobj: new(DateTime[TJira]),
			data: []byte(`"0001-01-01T00:00:00.000+0000"`),
			want: time.Time{},
		},
		{
			name: "TLayout",
			tobj: new(DateTime[TLayout]),
			data: []byte(`"01/02 03:04:05PM '06 -0700"`), // time.Layout
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name: "TANSIC",
			tobj: new(DateTime[TANSIC]),
			data: []byte(`"Mon Jan  2 15:04:05 2006"`), // time.ANSIC
			want: time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TUnixDate",
			tobj: new(DateTime[TUnixDate]),
			data: []byte(`"Mon Jan  2 15:04:05 MST 2006"`), // time.UnixDate
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", 0)),
		},
		{
			name: "TRubyDate",
			tobj: new(DateTime[TRubyDate]),
			data: []byte(`"Mon Jan 02 15:04:05 -0700 2006"`), // time.RubyDate
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name: "TRFC822",
			tobj: new(DateTime[TRFC822]),
			data: []byte(`"02 Jan 06 15:04 MST"`), // time.RFC822
			want: time.Date(2006, 1, 2, 15, 4, 0, 0, time.FixedZone("MST", 0)),
		},
		{
			name: "TRFC822Z",
			tobj: new(DateTime[TRFC822Z]),
			data: []byte(`"02 Jan 06 15:04 -0700"`), // time.RFC822Z
			want: time.Date(2006, 1, 2, 15, 4, 0, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name: "TRFC850",
			tobj: new(DateTime[TRFC850]),
			data: []byte(`"Monday, 02-Jan-06 15:04:05 MST"`), // time.RFC850
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", 0)),
		},
		{
			name: "TRFC1123",
			tobj: new(DateTime[TRFC1123]),
			data: []byte(`"Mon, 02 Jan 2006 15:04:05 MST"`), // time.RFC1123
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", 0)),
		},
		{
			name: "TRFC1123Z",
			tobj: new(DateTime[TRFC1123Z]),
			data: []byte(`"Mon, 02 Jan 2006 15:04:05 -0700"`), // time.RFC1123Z
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name: "TRFC3339",
			tobj: new(DateTime[TRFC3339]),
			data: []byte(`"2006-01-02T15:04:05-07:00"`), // time.RFC3339
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name: "TRFC3339Nano",
			tobj: new(DateTime[TRFC3339Nano]),
			data: []byte(`"2006-01-02T15:04:05.000-07:00"`), // time.RFC3339Nano
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name: "TKitchen",
			tobj: new(DateTime[TKitchen]),
			data: []byte(`"3:04PM"`), // time.Kitchen
			want: time.Date(0, time.January, 1, 15, 4, 0, 0, time.UTC),
		},
		{
			name: "TStamp",
			tobj: new(DateTime[TStamp]),
			data: []byte(`"Jan  2 15:04:05"`), // time.Stamp
			want: time.Date(0, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TStampMilli",
			tobj: new(DateTime[TStampMilli]),
			data: []byte(`"Jan  2 15:04:05.000"`), // time.StampMilli
			want: time.Date(0, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TStampMicro",
			tobj: new(DateTime[TStampMicro]),
			data: []byte(`"Jan  2 15:04:05.000000"`), // time.StampMicro
			want: time.Date(0, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TStampNano",
			tobj: new(DateTime[TStampNano]),
			data: []byte(`"Jan  2 15:04:05.000000000"`), // time.StampNano
			want: time.Date(0, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TDateTime",
			tobj: new(DateTime[TDateTime]),
			data: []byte(`"2006-01-02 15:04:05"`), // time.DateTime
			want: time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TDateOnly",
			tobj: new(DateTime[TDateOnly]),
			data: []byte(`"2006-01-02"`), // time.DateOnly
			want: time.Date(2006, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "TTimeOnly",
			tobj: new(DateTime[TTimeOnly]),
			data: []byte(`"15:04:05"`), // time.TimeOnly
			want: time.Date(0, time.January, 1, 15, 4, 5, 0, time.UTC),
		},
		{
			name: "TJira",
			tobj: new(DateTime[TJira]),
			data: []byte(`"2006-01-02T15:04:05.000-0700"`), // time.Jira
			want: time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dt, ok := tt.tobj.(json.Unmarshaler)
			require.True(t, ok, "tt.tobj does not implement json.Unmarshaler")
			require.NotNil(t, dt, "dt is nil")

			err := dt.UnmarshalJSON(tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				timeGetter, ok := dt.(interface{ Time() time.Time })

				require.True(t, ok, "dt does not implement Time() time.Time")
				require.Equal(t, tt.want, timeGetter.Time())
			}

			d := tt.tobj
			require.NotNil(t, d, "d is nil")

			err = json.Unmarshal(tt.data, &d)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				timeGetter, ok := d.(interface{ Time() time.Time })

				require.True(t, ok, "d does not implement Time() time.Time")
				require.Equal(t, tt.want, timeGetter.Time())
			}
		})
	}
}

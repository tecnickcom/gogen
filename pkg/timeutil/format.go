package timeutil

import "time"

// Marker types implementing DateTimeType for standard time layouts.

// TLayout selects [time.Layout].
type TLayout struct{}

// Format returns the layout string.
func (TLayout) Format() string { return time.Layout }

// TANSIC selects [time.ANSIC].
type TANSIC struct{}

// Format returns the layout string.
func (TANSIC) Format() string { return time.ANSIC }

// TUnixDate selects [time.UnixDate].
type TUnixDate struct{}

// Format returns the layout string.
func (TUnixDate) Format() string { return time.UnixDate }

// TRubyDate selects [time.RubyDate].
type TRubyDate struct{}

// Format returns the layout string.
func (TRubyDate) Format() string { return time.RubyDate }

// TRFC822 selects [time.RFC822].
type TRFC822 struct{}

// Format returns the layout string.
func (TRFC822) Format() string { return time.RFC822 }

// TRFC822Z selects [time.RFC822Z].
type TRFC822Z struct{}

// Format returns the layout string.
func (TRFC822Z) Format() string { return time.RFC822Z }

// TRFC850 selects [time.RFC850].
type TRFC850 struct{}

// Format returns the layout string.
func (TRFC850) Format() string { return time.RFC850 }

// TRFC1123 selects [time.RFC1123].
type TRFC1123 struct{}

// Format returns the layout string.
func (TRFC1123) Format() string { return time.RFC1123 }

// TRFC1123Z selects [time.RFC1123Z].
type TRFC1123Z struct{}

// Format returns the layout string.
func (TRFC1123Z) Format() string { return time.RFC1123Z }

// TRFC3339 selects [time.RFC3339].
type TRFC3339 struct{}

// Format returns the layout string.
func (TRFC3339) Format() string { return time.RFC3339 }

// TRFC3339Nano selects [time.RFC3339Nano].
type TRFC3339Nano struct{}

// Format returns the layout string.
func (TRFC3339Nano) Format() string { return time.RFC3339Nano }

// TKitchen selects [time.Kitchen].
type TKitchen struct{}

// Format returns the layout string.
func (TKitchen) Format() string { return time.Kitchen }

// TStamp selects [time.Stamp].
type TStamp struct{}

// Format returns the layout string.
func (TStamp) Format() string { return time.Stamp }

// TStampMilli selects [time.StampMilli].
type TStampMilli struct{}

// Format returns the layout string.
func (TStampMilli) Format() string { return time.StampMilli }

// TStampMicro selects [time.StampMicro].
type TStampMicro struct{}

// Format returns the layout string.
func (TStampMicro) Format() string { return time.StampMicro }

// TStampNano selects [time.StampNano].
type TStampNano struct{}

// Format returns the layout string.
func (TStampNano) Format() string { return time.StampNano }

// TDateTime selects [time.DateTime].
type TDateTime struct{}

// Format returns the layout string.
func (TDateTime) Format() string { return time.DateTime }

// TDateOnly selects [time.DateOnly].
type TDateOnly struct{}

// Format returns the layout string.
func (TDateOnly) Format() string { return time.DateOnly }

// TTimeOnly selects [time.TimeOnly].
type TTimeOnly struct{}

// Format returns the layout string.
func (TTimeOnly) Format() string { return time.TimeOnly }

// TimeJiraFormat is the Jira date-time format string.
const TimeJiraFormat = "2006-01-02T15:04:05.000-0700"

// TJira selects [TimeJiraFormat].
type TJira struct{}

// Format returns the layout string.
func (TJira) Format() string { return TimeJiraFormat }

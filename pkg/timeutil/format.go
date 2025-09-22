package timeutil

import "time"

// Common types for DateTime[T].

// TLayout represents the Layout time type.
type TLayout struct{}

// Format returns the Layout format string.
func (TLayout) Format() string { return time.Layout }

// TANSIC represents the ANSIC time type.
type TANSIC struct{}

// Format returns the ANSIC format string.
func (TANSIC) Format() string { return time.ANSIC }

// TUnixDate represents the UnixDate time type.
type TUnixDate struct{}

// Format returns the UnixDate format string.
func (TUnixDate) Format() string { return time.UnixDate }

// TRubyDate represents the RubyDate time type.
type TRubyDate struct{}

// Format returns the RubyDate format string.
func (TRubyDate) Format() string { return time.RubyDate }

// TRFC822 represents the RFC822 time type.
type TRFC822 struct{}

// Format returns the RFC822 format string.
func (TRFC822) Format() string { return time.RFC822 }

// TRFC822Z represents the RFC822Z time type.
type TRFC822Z struct{}

// Format returns the RFC822Z format string.
func (TRFC822Z) Format() string { return time.RFC822Z }

// TRFC850 represents the RFC850 time type.
type TRFC850 struct{}

// Format returns the RFC850 format string.
func (TRFC850) Format() string { return time.RFC850 }

// TRFC1123 represents the RFC1123 time type.
type TRFC1123 struct{}

// Format returns the RFC1123 format string.
func (TRFC1123) Format() string { return time.RFC1123 }

// TRFC1123Z represents the RFC1123Z time type.
type TRFC1123Z struct{}

// Format returns the RFC1123Z format string.
func (TRFC1123Z) Format() string { return time.RFC1123Z }

// TRFC3339 represents the RFC3339 time type.
type TRFC3339 struct{}

// Format returns the RFC3339 format string.
func (TRFC3339) Format() string { return time.RFC3339 }

// TRFC3339Nano represents the RFC3339Nano time type.
type TRFC3339Nano struct{}

// Format returns the RFC3339Nano format string.
func (TRFC3339Nano) Format() string { return time.RFC3339Nano }

// TKitchen represents the Kitchen time type.
type TKitchen struct{}

// Format returns the Kitchen format string.
func (TKitchen) Format() string { return time.Kitchen }

// TStamp represents the Stamp time type.
type TStamp struct{}

// Format returns the Stamp format string.
func (TStamp) Format() string { return time.Stamp }

// TStampMilli represents the StampMilli time type.
type TStampMilli struct{}

// Format returns the StampMilli format string.
func (TStampMilli) Format() string { return time.StampMilli }

// TStampMicro represents the StampMicro time type.
type TStampMicro struct{}

// Format returns the StampMicro format string.
func (TStampMicro) Format() string { return time.StampMicro }

// TStampNano represents the StampNano time type.
type TStampNano struct{}

// Format returns the StampNano format string.
func (TStampNano) Format() string { return time.StampNano }

// TDateTime represents the DateTime time type.
type TDateTime struct{}

// Format returns the DateTime format string.
func (TDateTime) Format() string { return time.DateTime }

// TDateOnly represents the DateOnly time type.
type TDateOnly struct{}

// Format returns the DateOnly format string.
func (TDateOnly) Format() string { return time.DateOnly }

// TTimeOnly represents the TimeOnly time type.
type TTimeOnly struct{}

// Format returns the TimeOnly format string.
func (TTimeOnly) Format() string { return time.TimeOnly }

// TimeJiraFormat is the Jira date-time format string.
const TimeJiraFormat = "2006-01-02T15:04:05.000-0700"

// TJira represents the Jira time type.
type TJira struct{}

// Format returns the Jira format string.
func (TJira) Format() string { return TimeJiraFormat }

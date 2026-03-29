/*
Package timeutil solves the recurring friction of marshaling time values to and
from JSON in Go services. The standard library's [time.Time] marshals as
RFC-3339 only, and [time.Duration] marshals as a raw nanosecond integer — both
mismatches for APIs that expect human-readable strings like "1h30m" or
"2023-01-02T15:04:05Z". This package provides two drop-in types that fix both
problems, plus a compile-time-safe mechanism for customizing the datetime format
without runtime configuration.

# Problem

Services that exchange time values over JSON face two common issues:

 1. Different API contracts require different datetime formats (RFC-3339,
    RFC-822, date-only, kitchen time, …). Switching formats means changing
    marshal/unmarshal logic scattered across the codebase, or wrapping
    [time.Time] with a custom type for every format.
 2. Duration values deserialised from JSON as raw nanosecond integers are
    unreadable in configuration files and API payloads. Human operators and
    API consumers expect strings like "30s" or "5m".

# Key Features

  - [DateTime]: a generic type `DateTime[T DateTimeType]` that wraps
    [time.Time] and implements [json.Marshaler] / [json.Unmarshaler] using the
    format string returned by the type parameter T. Switching formats is a
    one-character type-argument change — no runtime configuration, no string
    constants, caught by the compiler.
  - [DateTimeType]: the single-method interface (`Format() string`) that type
    parameters must satisfy, making it trivial to define custom formats.
  - Built-in format types: ready-to-use implementations of [DateTimeType] for
    every format constant in the standard library — [TRFC3339], [TRFC3339Nano],
    [TRFC822], [TRFC822Z], [TRFC850], [TRFC1123], [TRFC1123Z], [TUnixDate],
    [TANSIC], [TRubyDate], [TKitchen], [TStamp], [TStampMilli], [TStampMicro],
    [TStampNano], [TLayout], and [TDateOnly] / [TTimeOnly] — covering every
    use case without additional dependencies.
  - [Duration]: an alias for [time.Duration] that marshals as a human-readable
    string (e.g. "1h30m0s") and unmarshals from both string and numeric JSON
    values, making duration fields self-documenting in configuration files and
    API responses.

# Usage

Typed datetime in a struct:

	type Event struct {
	    StartedAt timeutil.DateTime[timeutil.TRFC3339]     `json:"started_at"`
	    EndedAt   timeutil.DateTime[timeutil.TDateOnly]    `json:"ended_at"`
	}

	e := Event{
	    StartedAt: timeutil.DateTime[timeutil.TRFC3339](time.Now()),
	}
	b, _ := json.Marshal(e)
	// {"started_at":"2023-01-02T15:04:05Z","ended_at":"2023-01-02"}

Human-readable duration in configuration:

	type Config struct {
	    Timeout timeutil.Duration `json:"timeout"`
	}

	var cfg Config
	_ = json.Unmarshal([]byte(`{"timeout":"30s"}`), &cfg)
	// cfg.Timeout is equivalent to 30 * time.Second

Custom format:

	type TYearMonth struct{}
	func (TYearMonth) Format() string { return "2006-01" }

	type Report struct {
	    Month timeutil.DateTime[TYearMonth] `json:"month"`
	}
*/
package timeutil

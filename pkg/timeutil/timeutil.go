/*
Package timeutil provides utility functions for working with time-related
operations.

This package offers a collection of functions that facilitate working with
time.Duration and time.Time types when marshaling and unmarshaling JSON
data. It includes a generic DateTime type that allows for flexible formatting
and parsing of time values based on specified layouts.

The DateTime type is defined as a generic type that wraps time.Time and
provides methods for JSON marshaling and unmarshaling using a specified time
format. The format is determined by the type parameter, which must implement
the DateTimeType interface. This interface requires a Format method that
returns the desired time format string.

Commonly used time formats are provided as types that implement the
DateTimeType interface.
*/
package timeutil

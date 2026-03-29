package errutil

import (
	"fmt"
	"runtime"
)

// Trace wraps err with caller file, line, and function metadata.
//
// It returns nil when err is nil. The original error is wrapped with %w so
// errors.Is and errors.As continue to work.
func Trace(err error) error {
	if err == nil {
		return nil
	}

	var (
		pc       uintptr
		file     string
		line     int
		ok       bool
		funcName string
	)

	pc, file, line, ok = runtime.Caller(1)
	if ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			funcName = fn.Name()
		}
	}

	return fmt.Errorf("file: %s, line: %d, function: %s, error: %w", file, line, funcName, err)
}

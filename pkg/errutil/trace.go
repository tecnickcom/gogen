package errutil

import (
	"fmt"
	"runtime"
)

// Trace wraps err with caller file, line, and function metadata.
//
// It returns nil when err is nil. The original error is wrapped with %w so
// errors.Is and errors.As continue to work. When caller metadata cannot be
// recovered, the function name falls back to "unknown".
//
// The file path is the one recorded by the compiler: it is absolute for a plain
// build and module-relative when the binary is built with -trimpath. Avoid
// relying on its exact form and prefer -trimpath for release builds to keep
// build-host paths out of error messages and logs.
func Trace(err error) error {
	if err == nil {
		return nil
	}

	pc, file, line, ok := runtime.Caller(1)

	funcName := "unknown"

	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = fn.Name()
		}
	}

	return fmt.Errorf("file: %s, line: %d, function: %s, error: %w", file, line, funcName, err)
}

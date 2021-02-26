// Package errtrace provides utilities to annotate errors.
package errtrace

import (
	"fmt"
	"runtime"
)

// Trace annotates the error message with the filename, line number and function name.
func Trace(err error) error {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("?:0 ?: %w", err)
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return fmt.Errorf("%s:%d ?: %w", file, line, err)
	}

	return fmt.Errorf("%s:%d %s: %w", file, line, fn.Name(), err)
}

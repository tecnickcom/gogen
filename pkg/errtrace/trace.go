/*
Package errtrace provides a function to automatically trace Go errors. Each
error is annotated with the filename, line number, and function name where it
was created.
*/
package errtrace

import (
	"fmt"
	"runtime"
)

// Trace annotates the error message with the filename, line number, and function name.
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

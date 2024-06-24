//go:build exttools

// Package exttools lists external build and test tools.
// These tools will appear in the `go.mod` file, but will not be a part of the build.
// They will be also excluded from the binaries as the "exttools" tag is not used.
package exttools

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "github.com/rakyll/gotest"
)

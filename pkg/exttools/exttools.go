//go:build exttools

/*
Package exttools lists external build and test tools. These tools will appear in
the `go.mod` file but will not be included in the build process. They will also
be excluded from the binaries since the "exttools" tag is not used.
*/
package exttools

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "github.com/rakyll/gotest"
)

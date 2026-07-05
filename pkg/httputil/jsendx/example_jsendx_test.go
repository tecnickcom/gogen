package jsendx_test

import (
	"fmt"
	"net/http"

	"github.com/tecnickcom/gogen/pkg/httputil/jsendx"
)

func ExampleWrap() {
	info := &jsendx.AppInfo{
		ProgramName:    "demo",
		ProgramVersion: "1.0.0",
		ProgramRelease: "1",
	}

	resp := jsendx.Wrap(http.StatusOK, info, "payload")

	// Only the deterministic fields are printed; DateTime and Timestamp vary per call.
	fmt.Println(resp.Program, resp.Version, resp.Release, resp.Code, resp.Message, resp.Data)

	// Output:
	// demo 1.0.0 1 200 OK payload
}

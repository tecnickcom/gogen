package main

import (
	"fmt"
)

func mockJSONMarshalError(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("SIMULATED json.Marshal ERROR")
}

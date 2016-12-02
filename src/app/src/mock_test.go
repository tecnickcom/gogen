package main

import (
	"fmt"
)

func mockJsonMarshalError(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("SIMULATED json.Marshal ERROR")
}

func mockJsonUnmarshalError(data []byte, v interface{}) error {
	return fmt.Errorf("SIMULATED json.Unmarshal ERROR")
}

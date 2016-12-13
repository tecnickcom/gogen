package main

import (
	"fmt"
	"io"
)

func mockJSONMarshalError(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("SIMULATED json.Marshal ERROR")
}

func mockJSONUnmarshalError(data []byte, v interface{}) error {
	return fmt.Errorf("SIMULATED json.Unmarshal ERROR")
}

func mockBase64DecodeString(s string) ([]byte, error) {
	return nil, fmt.Errorf("SIMULATED base64.StdEncoding.DecodeString ERROR")
}

func mockIoReadFullError(r io.Reader, buf []byte) (n int, err error) {
	return 0, fmt.Errorf("SIMULATED io.ReadFull ERROR")
}

func mockIoutilReadAll(r io.Reader) ([]byte, error) {
	return nil, fmt.Errorf("SIMULATED ioutil.ReadAll ERROR")
}

func mockSendJSONEncode(w io.Writer, v interface{}) error {
	return fmt.Errorf("SIMULATED sendJSONEncode ERROR")
}

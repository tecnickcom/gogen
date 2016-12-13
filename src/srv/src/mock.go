package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
)

// jsonMarshal is an alias for testing json.Marshal
var jsonMarshal = json.Marshal

// jsonUnmarshal is an alias for json.Unmarshal
var jsonUnmarshal = json.Unmarshal

// base64DecodeString is an alias for base64.StdEncoding.DecodeString
var base64DecodeString = base64.StdEncoding.DecodeString

// ioutilReadAll is an alias for ioutil.ReadAll
var ioutilReadAll = ioutil.ReadAll

// jsonHTTPEncode is a wrapper for json.NewEncoder(w).Encode(v)
func jsonHTTPEncode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

// sendJSONEncode is an alias for jsonHTTPEncode
var sendJSONEncode = jsonHTTPEncode

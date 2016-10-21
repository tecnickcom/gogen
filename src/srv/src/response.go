package main

// JSend status codes
const (
	StatusSuccess = "success"
	StatusFail    = "fail"
	StatusError   = "error"
)

// Response data format for HTTP
type Response struct {
	Program   string      `json:"program"`   // Program name
	Version   string      `json:"version"`   // Program version
	Release   string      `json:"release"`   // Program release number
	DateTime  string      `json:"datetime"`  // Human-readable date and time when the event occurred
	Timestamp int64       `json:"timestamp"` // Machine-readable UTC timestamp in nanoseconds since EPOCH
	Status    string      `json:"status"`    // Status code (error|fail|success)
	Code      int         `json:"code"`      // HTTP status code
	Message   string      `json:"message"`   // Error or status message
	Data      interface{} `json:"data"`      // Data payload
}

// convert the HTTP status code into JSend status
func getStatus(code int) string {
	if code >= 500 {
		return StatusError
	}
	if code >= 400 {
		return StatusFail
	}
	return StatusSuccess
}

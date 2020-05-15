package main

import (
	"testing"
)

func TestGetStatus(t *testing.T) {
	if getStatus(500) != StatusError {
		t.Errorf("The expected status error for the code 500 is: %s", StatusError)
	}
	if getStatus(400) != StatusFail {
		t.Errorf("The expected status error for the code 400 is: %s", StatusFail)
	}
	if getStatus(200) != StatusSuccess {
		t.Errorf("The expected status error for the code 200 is: %s", StatusSuccess)
	}
}

func BenchmarkGetStatus(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getStatus(200)
	}
}

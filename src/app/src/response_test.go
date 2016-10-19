package main

import (
	"fmt"
	"testing"
)

func TestGetStatus(t *testing.T) {
	if getStatus(500) != StatusError {
		t.Error(fmt.Errorf("The expected status error for the code 500 is: %s", StatusError))
	}
	if getStatus(400) != StatusFail {
		t.Error(fmt.Errorf("The expected status error for the code 400 is: %s", StatusFail))
	}
	if getStatus(200) != StatusSuccess {
		t.Error(fmt.Errorf("The expected status error for the code 200 is: %s", StatusSuccess))
	}
}

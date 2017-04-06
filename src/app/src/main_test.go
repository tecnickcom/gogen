package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"
)

func TestProgramVersion(t *testing.T) {
	os.Args = []string{ProgramName, "version"}
	out := getMainOutput(t)
	match, err := regexp.MatchString("^[\\d]+\\.[\\d]+\\.[\\d]+[\\s]*$", out)
	if err != nil {
		t.Error(fmt.Errorf("Unexpected error: %v", err))
	}
	if !match {
		t.Error(fmt.Errorf("The expected version has not been returned"))
	}
}

func getMainOutput(t *testing.T) string {
	old := os.Stdout // keep backup of the real stdout
	defer func() { os.Stdout = old }()
	r, w, err := os.Pipe()
	if err != nil {
		t.Error(fmt.Errorf("Unexpected error: %v", err))
	}
	os.Stdout = w

	// execute the main function
	main()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			t.Error(fmt.Errorf("Unexpected error: %v", err))
		}
		outC <- buf.String()
	}()

	// back to normal state
	err = w.Close()
	if err != nil {
		t.Error(fmt.Errorf("Unexpected error: %v", err))
	}
	out := <-outC

	return out
}

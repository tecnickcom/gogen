package main

import (
	"os"
	"reflect"
	"testing"
)

var badParamCases = []string{
	"--logLevel=",
	"--logLevel=INVALID",
}

func TestCliBadParamError(t *testing.T) {
	for _, param := range badParamCases {
		os.Args = []string{ProgramName, param}
		cmd, err := cli()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}
		if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
			t.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype)
			return
		}

		old := os.Stderr // keep backup of the real stdout
		defer func() { os.Stderr = old }()
		os.Stderr = nil

		// execute the main function
		if err := cmd.Execute(); err == nil {
			t.Errorf("An error was expected")
		}
	}
}

func TestWrongParamError(t *testing.T) {
	os.Args = []string{ProgramName, "--unknown"}
	_, err := cli()
	if err == nil {
		t.Errorf("An error was expected")
		return
	}
}

func TestCli(t *testing.T) {
	os.Args = []string{
		ProgramName,
		"--logLevel=debug",
		"--quantity=3",
		"--configDir=wrong/path",
	}
	cmd, err := cli()
	if err != nil {
		t.Errorf("An error wasn't expected: %v", err)
		return
	}
	if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
		t.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype)
		return
	}

	old := os.Stderr // keep backup of the real stdout
	defer func() { os.Stderr = old }()
	os.Stderr = nil
}

func TestCliConfigDir(t *testing.T) {
	os.Args = []string{
		ProgramName,
		"--configDir=resources/test/etc/~#PROJECT#~",
	}
	cmd, err := cli()
	if err != nil {
		t.Errorf("An error wasn't expected: %v", err)
		return
	}
	if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
		t.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype)
		return
	}

	old := os.Stderr // keep backup of the real stdout
	defer func() { os.Stderr = old }()
	os.Stderr = nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

var stopTestServerChan chan bool

var badParamCases = []string{
	"--logLevel=",
	"--logLevel=INVALID",
}

func TestCliBadParamError(t *testing.T) {
	for _, param := range badParamCases {
		os.Args = []string{ProgramName, param}
		cmd, err := cli()
		if err != nil {
			t.Error(fmt.Errorf("Unexpected error: %v", err))
			return
		}
		if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
			t.Error(fmt.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype))
			return
		}

		old := os.Stderr // keep backup of the real stdout
		defer func() { os.Stderr = old }()
		os.Stderr = nil

		// execute the main function
		if err := cmd.Execute(); err == nil {
			t.Error(fmt.Errorf("An error was expected"))
		}
	}
}

func TestWrongParamError(t *testing.T) {
	os.Args = []string{ProgramName, "--unknown"}
	_, err := cli()
	if err == nil {
		t.Error(fmt.Errorf("An error was expected"))
		return
	}
}

func TestCli(t *testing.T) {
	os.Args = []string{
		ProgramName,
		"--configDir=wrong/path",
	}
	cmd, err := cli()
	if err != nil {
		t.Error(fmt.Errorf("Unexpected error: %v", err))
		return
	}
	if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
		t.Error(fmt.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype))
		return
	}

	old := os.Stderr // keep backup of the real stdout
	defer func() { os.Stderr = old }()
	os.Stderr = nil

	// add an endpoint to test the panic handler
	routes = append(routes,
		Route{
			"GET",
			"/panic",
			triggerPanic,
			"TRIGGER PANIC",
		})
	defer func() { routes = routes[:len(routes)-1] }()

	// use two separate channels for server and client testing
	var twg sync.WaitGroup
	startTestServer(t, cmd, &twg)
	startTestClient(t)
	twg.Wait()
}

func startTestServer(t *testing.T, cmd *cobra.Command, twg *sync.WaitGroup) {

	stopTestServerChan = make(chan bool)

	twg.Add(1)
	go func() {
		defer twg.Done()

		chp := make(chan error, 1)
		go func() {
			chp <- cmd.Execute()
		}()

		stopped := false
		for {
			select {
			case err, ok := <-chp:
				if ok && !stopped && err != nil {
					stopTestServerChan <- true
					t.Error(fmt.Errorf("An error was not expected: %v", err))
				}
				return
			case <-stopTestServerChan:
				stopped = true
				stopServerChan <- os.Interrupt
			}
		}
	}()

	// wait for the server to shut down
	time.Sleep(5000 * time.Millisecond)
}

func startTestClient(t *testing.T) {

	// check if the server is running
	select {
	case stop, ok := <-stopTestServerChan:
		if ok && stop {
			return
		}
	default:
		break
	}

	defer func() { stopTestServerChan <- true }()

	testEndPoint(t, "GET", "/", "", 200)
	testEndPoint(t, "GET", "/status", "", 200)

	// error conditions

	testEndPoint(t, "GET", "/INVALID", "", 404) // NotFound
	testEndPoint(t, "DELETE", "/", "", 405)     // MethodNotAllowed
	testEndPoint(t, "GET", "/panic", "", 500)   // PanicHandler
}

// triggerPanic triggers a Panic
func triggerPanic(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	panic("TEST PANIC")
}

// isJSON returns true if the input is JSON
func isJSON(s []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(s, &js) == nil
}

func testEndPoint(t *testing.T, method string, path string, data string, code int) {
	var payload = []byte(data)
	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:8017%s", path), bytes.NewBuffer(payload))
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Error(fmt.Errorf("An error was not expected: %v", err))
			return
		}
	}()

	body, err := ioutilReadAll(resp.Body)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return
	}

	if resp.StatusCode != code {
		t.Error(fmt.Errorf("The expected '%s' status code is %d, found %d", path, code, resp.StatusCode))
		return
	}

	if !isJSON(body) {
		t.Error(fmt.Errorf("The body is not JSON"))
	}
}

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

var stopTestServerChan chan bool

var badParamCases = []string{
	"--logLevel=",
	"--logLevel=INVALID",
	"--configDir=../resources/test/etc/mysql_err",
	"--configDir=../resources/test/etc/tls_err",
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
	time.Sleep(6 * time.Second)
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

	testEndPoint(t, "GET", "/", "", "", 200)
	testEndPoint(t, "GET", "/ping", "", "", 200)
	testEndPoint(t, "GET", "/status", "", "", 503)
	testEndPoint(t, "GET", "/metrics", "", "", 200)

	testEndPoint(t, "GET", "/auth/refresh", "", "", 401)
	testEndPoint(t, "GET", "/auth/refresh", "", "ERROR", 401)
	testEndPoint(t, "GET", "/auth/refresh", "", "Bearer ", 401)
	testEndPoint(t, "GET", "/auth/refresh", "", "Bearer ERROR", 401)
	testEndPoint(t, "GET", "/auth/refresh", "", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QiLCJleHAiOjE1NzQzNTcwNjV9.imrC22sivbTLVgsaSDIL_GG9N6FDOkhl0S_BNobWxus", 401)

	testEndPoint(t, "POST", "/auth/login", "{\"username\":\"test\",\"password\":\"jwttest\"", "", 400)
	testEndPoint(t, "POST", "/auth/login", "{\"username\":\"error\",\"password\":\"jwttest\"}", "", 401)
	testEndPoint(t, "POST", "/auth/login", "{\"username\":\"test\",\"password\":\"ERROR\"}", "", 401)

	body := testEndPoint(t, "POST", "/auth/login", "{\"username\":\"test\",\"password\":\"jwttest\"}", "", 200)
	re := regexp.MustCompile(`"data":[\s]*"(.*)"`)
	token := re.FindSubmatch(body)
	testEndPoint(t, "GET", "/auth/refresh", "", "Bearer "+string(token[1]), 400)
	time.Sleep(time.Second)
	testEndPoint(t, "GET", "/auth/refresh", "", "Bearer "+string(token[1]), 200)

	testEndPoint(t, "GET", "/proxy/", "", "", 502)

	// error conditions

	testEndPoint(t, "GET", "/INVALID", "", "", 404) // NotFound
	testEndPoint(t, "DELETE", "/", "", "", 405)     // MethodNotAllowed
	testEndPoint(t, "GET", "/panic", "", "", 500)   // PanicHandler
}

// triggerPanic triggers a Panic
func triggerPanic(rw http.ResponseWriter, hr *http.Request) {
	panic("TEST PANIC")
}

// isJSON returns true if the input is JSON
func isJSON(s []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(s, &js) == nil
}

func testEndPoint(t *testing.T, method, path, data, token string, code int) []byte {
	var payload = []byte(data)
	req, err := http.NewRequest(method, fmt.Sprintf("https://127.0.0.1:8017%s", path), bytes.NewBuffer(payload))
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return nil
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	if len(token) > 0 {
		req.Header.Set("Authorization", token)
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	tr := &http.Transport{TLSClientConfig: tlsCfg}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return nil
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
		return nil
	}

	if resp.StatusCode != code {
		t.Error(fmt.Errorf("The expected '%s' status code is %d, found %d", path, code, resp.StatusCode))
		return nil
	}

	if path != "/metrics" && len(body) > 0 && !isJSON(body) {
		t.Error(fmt.Errorf("The body is not JSON: %v", body))
	}

	return body
}

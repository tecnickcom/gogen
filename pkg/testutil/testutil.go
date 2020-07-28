//go:generate mockgen -package mocks -destination ../internal/mocks/httpresp_mocks.go . TestHTTPResponseWriter

// Package testutil contains a set of utility functions used for testing
package testutil

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestHTTPResponseWriter wraps a standard lib http.ResponseWriter to allow mock generation
type TestHTTPResponseWriter interface {
	http.ResponseWriter
}

// Context returns a context initialized with a NOP logger for testing
func Context() context.Context {
	return logging.WithLogger(context.Background(), zap.NewNop())
}

// ContextWithLogObserver returns a context initialized with a NOP logger for testing
func ContextWithLogObserver(level zapcore.Level) (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	l := zap.New(core)
	return logging.WithLogger(context.Background(), l), logs
}

// ReplaceDateTime replaces a datetime. Useful to compare JSON responses, containing variable values
func ReplaceDateTime(src, repl string) string {
	re := regexp.MustCompile("([0-9]{4}\\-[0-9]{2}\\-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[^\"]*)")
	return re.ReplaceAllString(src, repl)
}

// ReplaceUnixTimestamp replaces a unix timestamp. Useful to compare JSON responses, containing variable values
func ReplaceUnixTimestamp(src, repl string) string {
	re := regexp.MustCompile("([0-9]{19})")
	return re.ReplaceAllString(src, repl)
}

// CaptureOutput hijacks and captures all log, stderr, stdout for testing
func CaptureOutput(fn func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
		log.SetOutput(os.Stderr)
	}()
	os.Stdout = writer
	os.Stderr = writer
	log.SetOutput(writer)

	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		_, _ = io.Copy(&buf, reader)
		out <- buf.String()
	}()
	wg.Wait()
	fn()
	_ = writer.Close()
	return <-out
}

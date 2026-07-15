package logsrv

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/tecnickcom/nurago/pkg/logutil"
)

// errWriter passes writes through to the destination and remembers the most recent failure, so Handle
// can return it: zerolog's Event API reports no error of its own. The error is still returned to
// zerolog as well, so its own fallback (a diagnostic on os.Stderr) remains the last resort for callers
// that discard Handle's error, such as slog.Logger.
//
// What it holds is a signal that the destination is failing, not a per-call status, and it is not
// possible to make it one: writes reach it serialized (it sits inside zerolog's SyncWriter) but takeErr
// runs outside that lock, so under concurrent logging a Handle whose own write succeeded may report a
// sibling's failure, and one whose write failed may find the error already taken by a sibling and
// return nil. Measured with a destination failing every write, about 72% of the failing Handle calls
// return an error and the rest return nil; a caller needing exact per-call attribution must write to the
// destination itself.
//
// Keeping the most recent failure rather than the first is deliberate: when a transient error (a full
// pipe) is followed by a fatal one, the fatal one is the useful diagnosis.
//
// Sequentially (one goroutine, which is how the error is normally consumed) it is exact: each failed
// write is reported to the next Handle, and a destination that recovers stops reporting the stale error.
type errWriter struct {
	w   io.Writer
	err atomic.Pointer[error]
}

// Write forwards to the destination, recording the error, if any, for takeErr. The error is boxed
// inside the failure branch, so the successful path (every path, normally) heap-allocates nothing:
// taking the address of the returned err itself would make it escape on every write.
func (ew *errWriter) Write(p []byte) (int, error) {
	n, err := ew.w.Write(p)
	if err != nil {
		boxed := err
		ew.err.Store(&boxed)
	}

	return n, err //nolint:wrapcheck // a transparent pass-through: zerolog handles (and reports) it.
}

// takeErr returns the pending write error and clears it, so a destination that recovers stops
// reporting the stale one (see errWriter for what "pending" guarantees under concurrency). A nil
// receiver reports no error, so a hand-built handler (as the tests construct) needs no destination.
func (ew *errWriter) takeErr() error {
	if ew == nil {
		return nil
	}

	boxed := ew.err.Swap(nil)
	if boxed == nil {
		return nil
	}

	return *boxed
}

// writerByFormat returns the zerolog output writer for the specified format (JSON, console, or discard).
func writerByFormat(f logutil.LogFormat, w io.Writer) io.Writer {
	switch f {
	case logutil.FormatJSON:
		return w
	case logutil.FormatConsole:
		// Colorize only when the destination is a terminal, so console output written
		// to a file or pipe does not embed raw ANSI escape sequences.
		noColor := !isTerminalWriter(w)

		return zerolog.ConsoleWriter{
			Out:     w,
			NoColor: noColor,
			// The timestamp is parsed with the same layout the handler writes (see timeLayout)
			// rather than with zerolog's process-global TimeFieldFormat, which its default
			// formatter would use: that global no longer describes the field, so a binary that
			// changed it (to a Unix format, say) would make every console line fall back to
			// printing the raw RFC 3339 timestamp instead of the console time format.
			FormatTimestamp: consoleTimestamp(noColor),
		}
	case logutil.FormatNone:
		return io.Discard
	default:
		return w
	}
}

// consoleTimeFormat is the time format the console renders (zerolog's ConsoleWriter default).
const consoleTimeFormat = time.Kitchen

// noColorEnv is the environment variable (see https://no-color.org) zerolog's own colorize helper
// consults, per call, before emitting any escape sequence.
const noColorEnv = "NO_COLOR"

// darkGray wraps s in the ANSI escape zerolog's console uses for the timestamp. Its own colorize
// helper is unexported, so the sequence is written here.
func darkGray(s string) string { return "\x1b[90m" + s + "\x1b[0m" }

// consoleTimestamp returns the ConsoleWriter timestamp formatter: it parses the field with the same
// layout the handler writes (timeLayoutBare) and renders it in the local zone in the console time
// format, mirroring zerolog's own formatter, including its dark-gray coloring (suppressed by the
// NO_COLOR environment variable, which zerolog reads per call, as here), and its "<nil>" for a
// missing field so a record with a zero time reads as before, minus the dependency on the
// process-global zerolog.TimeFieldFormat, which no longer describes the field.
func consoleTimestamp(noColor bool) zerolog.Formatter {
	return func(i any) string {
		out := consoleTimeValue(i)

		if noColor || os.Getenv(noColorEnv) != "" {
			return out
		}

		return darkGray(out)
	}
}

// consoleTimeValue renders the decoded time field for the console. A value that is not the timestamp
// this handler writes cannot be a record timestamp: it is a user attribute colliding with the
// reserved "time" key, or a missing field, so it is passed through rather than dropped: an
// unparseable string verbatim, any other value via its default formatting, and a missing field as
// zerolog's own "<nil>" placeholder.
func consoleTimeValue(i any) string {
	s, ok := i.(string)
	if !ok {
		if i == nil {
			return "<nil>"
		}

		return fmt.Sprint(i)
	}

	ts, err := time.Parse(timeLayoutBare, s)
	if err != nil {
		return s
	}

	// The console is read by a human at a local terminal, so the instant is shown in the local zone
	// (as zerolog's own formatter does): time.Kitchen carries no zone marker, and rendering each
	// record in whatever zone it was stamped with would make lines look out of order.
	return ts.In(time.Local).Format(consoleTimeFormat) //nolint:gosmopolitan // deliberate: see above.
}

// isTerminalWriter reports whether w is a terminal (character device). Non-terminal
// writers (files, pipes, in-memory buffers) return false so console output is emitted
// without color escapes.
//
// It only recognizes a bare *os.File: a terminal wrapped in a decorator (e.g. a
// bufio.Writer) is treated as non-terminal and rendered without color. This is a
// deliberate, dependency-free heuristic (golang.org/x/term is not an allowed import);
// callers needing precise control should pass the terminal *os.File directly.
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

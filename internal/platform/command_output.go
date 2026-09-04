package platform

import (
	"bytes"
	"unicode/utf8"
)

// commandOutputLimit is the retained tail size for each captured stream.
const commandOutputLimit = 64 * 1024

// WithBoundedOutput limits captured stdout and stderr to their final 64 KiB.
// Streaming writers still receive all output. Custom Runner implementations are
// unchanged; callers that parse complete command output should use the default.
func WithBoundedOutput(runner Runner) Runner {
	switch execRunner := runner.(type) {
	case ExecRunner:
		execRunner.boundedOutput = true
		return execRunner
	case *ExecRunner:
		copy := *execRunner
		copy.boundedOutput = true
		return copy
	default:
		return runner
	}
}

// commandOutputCapture is written by a single subprocess stream's copy goroutine.
// Its circular tail never grows with total command output.
type commandOutputCapture struct {
	bounded   bool
	full      bytes.Buffer
	tail      []byte
	start     int
	used      int
	truncated bool
}

func (capture *commandOutputCapture) Write(p []byte) (int, error) {
	if !capture.bounded {
		return capture.full.Write(p)
	}
	n := len(p)
	if n == 0 {
		return 0, nil
	}
	if capture.tail == nil {
		capture.tail = make([]byte, commandOutputLimit)
	}
	if n > commandOutputLimit-capture.used {
		capture.truncated = true
	}
	if n >= commandOutputLimit {
		copy(capture.tail, p[n-commandOutputLimit:])
		capture.start, capture.used = 0, commandOutputLimit
		return n, nil
	}
	end := (capture.start + capture.used) % commandOutputLimit
	first := copy(capture.tail[end:], p)
	copy(capture.tail, p[first:])
	if overflow := capture.used + n - commandOutputLimit; overflow > 0 {
		capture.start = (capture.start + overflow) % commandOutputLimit
		capture.used = commandOutputLimit
	} else {
		capture.used += n
	}
	return n, nil
}

func (capture *commandOutputCapture) String() string {
	if !capture.bounded {
		return capture.full.String()
	}
	retained := make([]byte, capture.used)
	first := copy(retained, capture.tail[capture.start:])
	copy(retained[first:], capture.tail[:capture.start])
	// A byte-limited tail can start inside a UTF-8 encoding. Drop only the
	// continuation bytes at that cut so otherwise valid output stays valid.
	if capture.truncated {
		for len(retained) > 0 && !utf8.RuneStart(retained[0]) {
			retained = retained[1:]
		}
	}
	return string(retained)
}

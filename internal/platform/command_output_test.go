package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func outputHelperArgs(scenario string, size int) []string {
	return []string{"-test.run=TestOutputHelperProcess", "--", scenario, strconv.Itoa(size)}
}

func TestOutputHelperProcess(t *testing.T) {
	if len(os.Args) < 4 || os.Args[len(os.Args)-3] != "--" {
		return
	}
	scenario := os.Args[len(os.Args)-2]
	size, err := strconv.Atoi(os.Args[len(os.Args)-1])
	if err != nil {
		os.Exit(99)
	}
	chunk := strings.Repeat("é界🙂", 1024)
	for written := 0; written < size; written += len(chunk) {
		fmt.Fprint(os.Stdout, chunk)
		fmt.Fprint(os.Stderr, chunk)
	}
	fmt.Fprint(os.Stdout, "\nstdout finished\n")
	fmt.Fprint(os.Stderr, "\nstderr finished\n")
	switch scenario {
	case "failure":
		os.Exit(7)
	case "cancel":
		fmt.Fprint(os.Stdout, "READY\n")
		for {
			time.Sleep(time.Second)
		}
	default:
		os.Exit(0)
	}
}

func TestExecRunnerBoundedOutputRetainsUnicodeTailsAndStreamsEverything(t *testing.T) {
	const size = 2 * 1024 * 1024
	full, err := NewExecRunner().Run(context.Background(), os.Args[0], outputHelperArgs("success", size)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Stdout) < size || len(full.Stderr) < size || full.StdoutTruncated || full.StderrTruncated {
		t.Fatal("default capture lost output")
	}
	for _, scenario := range []string{"success", "failure"} {
		t.Run(scenario, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			verbose := NewVerboseExecRunner(&stdout, &stderr)
			result, err := WithBoundedOutput(&verbose).Run(context.Background(), os.Args[0], outputHelperArgs(scenario, size)...)
			if scenario == "failure" {
				var commandErr CommandError
				if !errors.As(err, &commandErr) || result.ExitCode != 7 {
					t.Fatalf("failure = %+v, %v", result, err)
				}
				if !commandErr.Result.StdoutTruncated || !commandErr.Result.StderrTruncated {
					t.Fatal("error lost truncation metadata")
				}
				for _, stream := range []string{"stdout", "stderr"} {
					if !strings.Contains(err.Error(), stream+":\n... output truncated; showing captured tail") {
						t.Fatalf("missing truncation marker: %v", err)
					}
				}
			} else if err != nil {
				t.Fatal(err)
			}
			for _, pair := range [][2]string{{result.Stdout, full.Stdout}, {result.Stderr, full.Stderr}} {
				if len(pair[0]) > commandOutputLimit || len(pair[0]) < commandOutputLimit-utf8.UTFMax || !utf8.ValidString(pair[0]) || !strings.HasSuffix(pair[1], pair[0]) {
					t.Fatalf("incorrect retained tail: len=%d, valid=%v", len(pair[0]), utf8.ValidString(pair[0]))
				}
			}
			if !result.StdoutTruncated || !result.StderrTruncated {
				t.Fatal("missing truncation flags")
			}
			if !strings.HasSuffix(stdout.String(), full.Stdout) || stderr.String() != full.Stderr {
				t.Fatal("verbose streaming was truncated")
			}
			if verbose.boundedOutput {
				t.Fatal("option mutated original pointer")
			}
		})
	}
}

type cancelAfterOutput struct {
	cancel context.CancelFunc
	once   sync.Once
	tail   string
}

func (writer *cancelAfterOutput) Write(p []byte) (int, error) {
	text := writer.tail + string(p)
	if strings.Contains(text, "READY\n") {
		writer.once.Do(writer.cancel)
	}
	if len(text) > 6 {
		text = text[len(text)-6:]
	}
	writer.tail = text
	return len(p), nil
}

func TestExecRunnerBoundedOutputPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	writer := &cancelAfterOutput{cancel: cancel}
	result, err := WithBoundedOutput(NewVerboseExecRunner(writer, io.Discard)).Run(ctx, os.Args[0], outputHelperArgs("cancel", 2*commandOutputLimit)...)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want cancellation", err)
	}
	if !result.StdoutTruncated || !result.StderrTruncated || !strings.HasSuffix(result.Stdout, "READY\n") {
		t.Fatal("cancellation lost captured tail or flags")
	}
}

func TestCommandOutputCaptureByteCapsAcrossWriteBoundaries(t *testing.T) {
	for _, size := range []int{0, commandOutputLimit - 1, commandOutputLimit, commandOutputLimit + 1, 3*commandOutputLimit + 17} {
		for _, chunkSize := range []int{1, 137, 3 * commandOutputLimit} {
			t.Run(fmt.Sprintf("size-%d/chunk-%d", size, chunkSize), func(t *testing.T) {
				data := bytes.Repeat([]byte("abcde"), (size+4)/5)[:size]
				capture := commandOutputCapture{bounded: true}
				for offset := 0; offset < size; offset += chunkSize {
					end := min(size, offset+chunkSize)
					n, err := capture.Write(data[offset:end])
					if n != end-offset || err != nil {
						t.Fatalf("Write = %d, %v", n, err)
					}
				}
				want := data[max(0, size-commandOutputLimit):]
				if capture.String() != string(want) || capture.truncated != (size > commandOutputLimit) {
					t.Fatalf("tail length=%d truncated=%v; want length=%d truncated=%v", len(capture.String()), capture.truncated, len(want), size > commandOutputLimit)
				}
			})
		}
	}
}

func TestWithBoundedOutputPreservesRunnerOptions(t *testing.T) {
	isolation := false
	original := NewVerboseExecRunner(io.Discard, io.Discard)
	original.trustCommandPath, original.preserveUserPath = true, true
	original.waitDelay = 123 * time.Millisecond
	original.processGroupIsolation = &isolation
	for _, input := range []Runner{original, &original} {
		got := WithBoundedOutput(input).(ExecRunner)
		if !got.boundedOutput || !got.trustCommandPath || !got.preserveUserPath || !got.renderCommands || got.waitDelay != original.waitDelay || got.processGroupIsolation != &isolation || got.stdout != original.stdout || got.stderr != original.stderr {
			t.Fatal("bounded variant dropped runner options")
		}
	}
	custom := &outputTestRunner{}
	if WithBoundedOutput(custom) != custom {
		t.Fatal("custom runner changed")
	}
}

type outputTestRunner struct{}

func (*outputTestRunner) Run(context.Context, string, ...string) (CommandResult, error) {
	return CommandResult{}, nil
}

func TestCommandOutputSummarySkipsBlankLinesAndBoundsUnicodePrefix(t *testing.T) {
	output := "old\r\n\n" + strings.Repeat(" \r\n", 4)
	for i := 0; i < 10; i++ {
		output += fmt.Sprintf("line%d\r\n\n", i)
	}
	summary := commandOutputSection("stdout", output, false)
	if strings.Contains(summary, "old") || !strings.Contains(summary, "line0\nline1") || !strings.HasSuffix(summary, "line9") {
		t.Fatalf("summary = %q", summary)
	}
	line := strings.Repeat("界", 100000)
	truncated := truncateCommandOutputLine(line)
	if truncated != strings.Repeat("界", commandOutputMaxLineLength)+"..." || !utf8.ValidString(truncated) {
		t.Fatal("incorrect Unicode line prefix")
	}
	if got := commandOutputSection("stderr", " \r\n", false); got != "" {
		t.Fatalf("blank summary = %q", got)
	}
	if got := commandOutputSection("stderr", "", true); !strings.Contains(got, "output truncated") {
		t.Fatal("empty tail lost truncation marker")
	}
}

var benchmarkCapturedResult CommandResult
var benchmarkOutputSummary string

func BenchmarkExecRunnerOutputCapture(b *testing.B) {
	for _, size := range []int{64 * 1024, 8 * 1024 * 1024} {
		for _, bounded := range []bool{false, true} {
			b.Run(fmt.Sprintf("bytes-%d/bounded-%v", size, bounded), func(b *testing.B) {
				var runner Runner = NewExecRunner()
				if bounded {
					runner = WithBoundedOutput(runner)
				}
				b.ReportAllocs()
				b.SetBytes(int64(size * 2))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					result, err := runner.Run(context.Background(), os.Args[0], outputHelperArgs("success", size)...)
					if err != nil {
						b.Fatal(err)
					}
					benchmarkCapturedResult = result
				}
			})
		}
	}
}

func BenchmarkCommandErrorOutputFormatting(b *testing.B) {
	for _, size := range []int{1024 * 1024, 64 * 1024 * 1024} {
		for _, giantLine := range []bool{false, true} {
			b.Run(fmt.Sprintf("bytes-%d/giant-line-%v", size, giantLine), func(b *testing.B) {
				unit := "x\n"
				if giantLine {
					unit = "xx"
				}
				commandErr := CommandError{Result: CommandResult{Name: "test", Stderr: strings.Repeat(unit, size/2)}, Err: errors.New("failed")}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					benchmarkOutputSummary = commandErr.Error()
				}
			})
		}
	}
}

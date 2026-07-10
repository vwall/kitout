//go:build darwin || linux

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestExecRunnerInteractiveCommandOwnsForegroundTerminal(t *testing.T) {
	if !processHasControllingTerminal() {
		t.Skip("test requires a controlling terminal")
	}

	runner := NewExecRunner()
	if _, err := runner.Run(context.Background(), os.Args[0], processTreeHelperArgs("check-foreground-terminal")...); err != nil {
		t.Fatalf("Run foreground terminal helper: %v", err)
	}
}

func TestExecRunnerInterruptsForegroundCommandThatIgnoresSIGINT(t *testing.T) {
	if !processHasControllingTerminal() {
		t.Skip("test requires a controlling terminal")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runner := NewExecRunner()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("ignore-interrupt-and-signal-group")...)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after its foreground process group received SIGINT")
	}
}

func TestExecRunnerInterruptsForegroundCommandThatIgnoresSIGQUITAndSIGINT(t *testing.T) {
	if !processHasControllingTerminal() {
		t.Skip("test requires a controlling terminal")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runner := NewExecRunner()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("ignore-quit-and-interrupt-and-signal-group")...)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after its foreground process group received SIGQUIT")
	}
}

func TestExecRunnerRestoresForegroundCommandAfterTerminalResume(t *testing.T) {
	if !processHasControllingTerminal() {
		t.Skip("test requires ownership of a controlling terminal")
	}

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGTSTP)
	defer signal.Stop(stopped)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewExecRunner()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("ignore-stop-and-check-resumed-foreground")...)
		done <- err
	}()

	select {
	case <-stopped:
	case err := <-done:
		t.Fatalf("Run returned before forwarding SIGTSTP: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("foreground process group did not forward SIGTSTP")
	}
	select {
	case err := <-done:
		t.Fatalf("SIGTSTP-ignoring command kept running while Kitout was suspended: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	setTestTerminalForegroundProcessGroup(t, syscall.Getpgrp())
	if err := syscall.Kill(os.Getpid(), syscall.SIGCONT); err != nil {
		t.Fatalf("resume runner: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run resumed foreground helper: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not resume its command in the foreground")
	}
}

func TestExecRunnerDoesNotStealTerminalAfterBackgroundResume(t *testing.T) {
	if !processHasControllingTerminal() {
		t.Skip("test requires ownership of a controlling terminal")
	}

	parentProcessGroup := syscall.Getpgrp()
	shellOwner := exec.Command(os.Args[0], processTreeHelperArgs("block")...)
	shellOwner.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := shellOwner.Start(); err != nil {
		t.Fatalf("start shell foreground owner: %v", err)
	}
	shellProcessGroup := shellOwner.Process.Pid
	t.Cleanup(func() {
		setTestTerminalForegroundProcessGroup(t, parentProcessGroup)
		terminateTestProcessGroup(shellProcessGroup)
		_ = shellOwner.Wait()
	})

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGTSTP)
	defer signal.Stop(stopped)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewExecRunner()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("ignore-stop-and-exit-after-background-resume")...)
		done <- err
	}()

	select {
	case <-stopped:
	case err := <-done:
		t.Fatalf("Run returned before forwarding SIGTSTP: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("foreground process group did not forward SIGTSTP")
	}
	select {
	case err := <-done:
		t.Fatalf("SIGTSTP-ignoring command kept running while Kitout was suspended: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	setTestTerminalForegroundProcessGroup(t, shellProcessGroup)
	if err := syscall.Kill(os.Getpid(), syscall.SIGCONT); err != nil {
		t.Fatalf("resume runner in background: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run resumed background helper: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not finish its background command")
	}
	if foregroundProcessGroup := testTerminalForegroundProcessGroup(t); foregroundProcessGroup != shellProcessGroup {
		t.Fatalf("foreground process group = %d, want shell owner %d", foregroundProcessGroup, shellProcessGroup)
	}
}

func TestExecRunnerMapsInterruptSignalToContextCancellation(t *testing.T) {
	runner := NewExecRunner()
	isolateProcessGroup := true
	runner.processGroupIsolation = &isolateProcessGroup

	_, err := runner.Run(context.Background(), os.Args[0], processTreeHelperArgs("interrupt-self")...)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
}

func TestExecRunnerMapsConventionalInterruptExitToContextCancellation(t *testing.T) {
	runner := NewExecRunner()
	isolateProcessGroup := true
	runner.processGroupIsolation = &isolateProcessGroup

	_, err := runner.Run(context.Background(), os.Args[0], processTreeHelperArgs("exit-130")...)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
}

func TestExecRunnerCancellationTerminatesProcessTree(t *testing.T) {
	markerPath := filepath.Join(t.TempDir(), "descendant.pid")
	runner := NewExecRunner()
	runner.waitDelay = 5 * time.Second
	isolateProcessGroup := true
	runner.processGroupIsolation = &isolateProcessGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("wait-descendant", markerPath)...)
		result <- err
	}()

	descendantPID := waitForHelperPID(t, markerPath)
	t.Cleanup(func() { terminateTestProcess(descendantPID) })
	cancel()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("Run returned nil error after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return promptly after cancellation")
	}

	waitForTestProcessExit(t, descendantPID)
}

func TestExecRunnerCancellationTerminatesNonRelayingTerminalTree(t *testing.T) {
	markerPath := filepath.Join(t.TempDir(), "descendant.pid")
	runner := NewExecRunner()
	runner.waitDelay = 5 * time.Second
	isolateProcessGroup := false
	runner.processGroupIsolation = &isolateProcessGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("wait-descendant", markerPath)...)
		result <- err
	}()

	descendantPID := waitForHelperPID(t, markerPath)
	t.Cleanup(func() { terminateTestProcess(descendantPID) })
	cancel()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("Run returned nil error after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not terminate the non-relaying process tree")
	}

	waitForTestProcessExit(t, descendantPID)
}

func TestExecRunnerCancellationGracefullySignalsEntireProcessGroup(t *testing.T) {
	tempDir := t.TempDir()
	pidMarker := filepath.Join(tempDir, "descendant.pid")
	termMarker := filepath.Join(tempDir, "descendant.term")
	runner := NewExecRunner()
	runner.waitDelay = 5 * time.Second
	isolateProcessGroup := true
	runner.processGroupIsolation = &isolateProcessGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("wait-signal-descendant", pidMarker, termMarker)...)
		result <- err
	}()

	descendantPID := waitForHelperPID(t, pidMarker)
	t.Cleanup(func() { terminateTestProcess(descendantPID) })
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not terminate the process group")
	}
	if _, err := os.Stat(termMarker); err != nil {
		t.Fatalf("graceful SIGTERM marker: %v", err)
	}
}

func TestExecRunnerCancellationFreezesForkingTerminalTree(t *testing.T) {
	tempDir := t.TempDir()
	markerDir := filepath.Join(tempDir, "pids")
	if err := os.Mkdir(markerDir, 0o755); err != nil {
		t.Fatalf("create PID marker directory: %v", err)
	}
	rootMarkerPath := filepath.Join(tempDir, "root.pid")
	runner := NewExecRunner()
	runner.waitDelay = 5 * time.Second
	isolateProcessGroup := false
	runner.processGroupIsolation = &isolateProcessGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("fork-descendants", rootMarkerPath, markerDir, "32")...)
		result <- err
	}()

	processGroupID := waitForHelperPID(t, rootMarkerPath)
	t.Cleanup(func() { terminateTestProcessGroup(processGroupID) })
	_ = waitForHelperPIDs(t, markerDir, 8)
	cancel()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("Run returned nil error after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not terminate the forking process tree")
	}

	waitForTestProcessGroupExit(t, processGroupID)
	for _, pid := range helperPIDs(t, markerDir) {
		waitForTestProcessExit(t, pid)
	}
}

func TestExecRunnerCleanupTerminatesOrphanAfterLeaderExit(t *testing.T) {
	childMarker := filepath.Join(t.TempDir(), "child.pid")
	runner := NewExecRunner()
	runner.waitDelay = 5 * time.Second
	isolateProcessGroup := false
	runner.processGroupIsolation = &isolateProcessGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, os.Args[0], processTreeHelperArgs("orphan-pipe", childMarker)...)
		result <- err
	}()

	childPID := waitForHelperPID(t, childMarker)
	t.Cleanup(func() { terminateTestProcess(childPID) })
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not terminate the orphaned descendant after leader exit")
	}
	waitForTestProcessExit(t, childPID)
}

func TestExecRunnerBoundsWaitForInheritedPipes(t *testing.T) {
	markerPath := filepath.Join(t.TempDir(), "descendant.pid")
	runner := NewExecRunner()
	runner.waitDelay = 50 * time.Millisecond
	type runResponse struct {
		result CommandResult
		err    error
	}
	done := make(chan runResponse, 1)
	go func() {
		result, err := runner.Run(context.Background(), os.Args[0], processTreeHelperArgs("orphan-pipe", markerPath)...)
		done <- runResponse{result: result, err: err}
	}()

	descendantPID := waitForHelperPID(t, markerPath)
	t.Cleanup(func() { terminateTestProcess(descendantPID) })

	select {
	case response := <-done:
		if !errors.Is(response.err, exec.ErrWaitDelay) {
			t.Fatalf("Run error = %v, want exec.ErrWaitDelay", response.err)
		}
		if response.result.ExitCode != 0 {
			t.Fatalf("ExitCode = %d, want 0", response.result.ExitCode)
		}
		waitForTestProcessExit(t, descendantPID)
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not bound its wait for inherited pipes")
	}
}

func TestProcessTreeHelper(t *testing.T) {
	scenario, args, ok := processTreeHelperScenario(os.Args)
	if !ok {
		return
	}

	switch scenario {
	case "wait-descendant", "orphan-pipe":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "process-tree helper requires a marker path")
			os.Exit(2)
		}
		descendant := exec.Command(os.Args[0], processTreeHelperArgs("block")...)
		descendant.Stdout = os.Stdout
		descendant.Stderr = os.Stderr
		if err := descendant.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "start descendant: %v\n", err)
			os.Exit(2)
		}
		if err := os.WriteFile(args[0], []byte(strconv.Itoa(descendant.Process.Pid)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write descendant pid: %v\n", err)
			_ = descendant.Process.Kill()
			os.Exit(2)
		}
		if scenario == "orphan-pipe" {
			os.Exit(0)
		}
		if err := descendant.Wait(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "fork-descendants":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "fork-descendants helper requires a root marker, PID directory, and child limit")
			os.Exit(2)
		}
		if err := os.WriteFile(args[0], []byte(strconv.Itoa(syscall.Getpgrp())), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write helper process group: %v\n", err)
			os.Exit(2)
		}
		descendant := exec.Command(os.Args[0], processTreeHelperArgs("fork-bounded", args[1], args[2])...)
		descendant.Stdout = os.Stdout
		descendant.Stderr = os.Stderr
		if err := descendant.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "start forking descendant: %v\n", err)
			os.Exit(2)
		}
		if err := descendant.Wait(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "wait-signal-descendant":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "wait-signal-descendant helper requires PID and SIGTERM marker paths")
			os.Exit(2)
		}
		descendant := exec.Command(os.Args[0], processTreeHelperArgs("observe-term", args[0], args[1])...)
		descendant.Stdout = os.Stdout
		descendant.Stderr = os.Stderr
		if err := descendant.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "start signal-observing descendant: %v\n", err)
			os.Exit(2)
		}
		if err := descendant.Wait(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "fork-bounded":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "fork-bounded helper requires a marker directory and child limit")
			os.Exit(2)
		}
		limit, err := strconv.Atoi(args[1])
		if err != nil || limit < 1 || limit > 128 {
			fmt.Fprintf(os.Stderr, "invalid fork-bounded child limit %q\n", args[1])
			os.Exit(2)
		}
		writeHelperPIDMarker(args[0], os.Getpid())
		for i := 0; i < limit; i++ {
			descendant := exec.Command(os.Args[0], processTreeHelperArgs("block")...)
			descendant.Stdout = os.Stdout
			descendant.Stderr = os.Stderr
			if err := descendant.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "start blocking descendant: %v\n", err)
				os.Exit(2)
			}
			writeHelperPIDMarker(args[0], descendant.Process.Pid)
			time.Sleep(time.Millisecond)
		}
		for {
			time.Sleep(time.Second)
		}
	case "block":
		if len(args) == 1 {
			if err := os.WriteFile(args[0], []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "write blocking helper pid: %v\n", err)
				os.Exit(2)
			}
		}
		for {
			time.Sleep(time.Second)
		}
	case "check-foreground-terminal":
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open controlling terminal: %v\n", err)
			os.Exit(2)
		}
		var foregroundProcessGroup int
		_, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL,
			tty.Fd(),
			uintptr(syscall.TIOCGPGRP),
			uintptr(unsafe.Pointer(&foregroundProcessGroup)),
		)
		_ = tty.Close()
		if errno != 0 {
			fmt.Fprintf(os.Stderr, "read terminal foreground process group: %v\n", errno)
			os.Exit(2)
		}
		if foregroundProcessGroup != syscall.Getpgrp() {
			fmt.Fprintf(os.Stderr, "foreground process group = %d, helper process group = %d\n", foregroundProcessGroup, syscall.Getpgrp())
			os.Exit(2)
		}
	case "interrupt-self":
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
		time.Sleep(time.Second)
		os.Exit(2)
	case "ignore-interrupt-and-signal-group":
		signal.Ignore(syscall.SIGINT)
		if err := syscall.Kill(-syscall.Getpgrp(), syscall.SIGINT); err != nil {
			fmt.Fprintf(os.Stderr, "signal foreground process group: %v\n", err)
			os.Exit(2)
		}
		for {
			time.Sleep(time.Second)
		}
	case "ignore-quit-and-interrupt-and-signal-group":
		signal.Ignore(syscall.SIGQUIT, syscall.SIGINT)
		if err := syscall.Kill(-syscall.Getpgrp(), syscall.SIGQUIT); err != nil {
			fmt.Fprintf(os.Stderr, "signal foreground process group: %v\n", err)
			os.Exit(2)
		}
		for {
			time.Sleep(time.Second)
		}
	case "ignore-stop-and-check-resumed-foreground":
		signal.Ignore(syscall.SIGTSTP)
		if err := syscall.Kill(-syscall.Getpgrp(), syscall.SIGTSTP); err != nil {
			fmt.Fprintf(os.Stderr, "stop foreground process group: %v\n", err)
			os.Exit(2)
		}
		time.Sleep(100 * time.Millisecond)
		assertHelperOwnsForegroundTerminal()
	case "ignore-stop-and-exit-after-background-resume":
		signal.Ignore(syscall.SIGTSTP)
		if err := syscall.Kill(-syscall.Getpgrp(), syscall.SIGTSTP); err != nil {
			fmt.Fprintf(os.Stderr, "stop foreground process group: %v\n", err)
			os.Exit(2)
		}
		time.Sleep(100 * time.Millisecond)
	case "exit-130":
		os.Exit(130)
	case "observe-term":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "observe-term helper requires PID and SIGTERM marker paths")
			os.Exit(2)
		}
		term := make(chan os.Signal, 1)
		signal.Notify(term, syscall.SIGTERM)
		defer signal.Stop(term)
		if err := os.WriteFile(args[0], []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write signal observer pid: %v\n", err)
			os.Exit(2)
		}
		<-term
		if err := os.WriteFile(args[1], nil, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write SIGTERM marker: %v\n", err)
			os.Exit(2)
		}
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown process-tree helper scenario %q\n", scenario)
		os.Exit(2)
	}
}

func assertHelperOwnsForegroundTerminal() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open controlling terminal: %v\n", err)
		os.Exit(2)
	}
	foregroundProcessGroup, err := terminalForegroundProcessGroup(tty)
	_ = tty.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read terminal foreground process group: %v\n", err)
		os.Exit(2)
	}
	if foregroundProcessGroup != syscall.Getpgrp() {
		fmt.Fprintf(os.Stderr, "foreground process group = %d, helper process group = %d\n", foregroundProcessGroup, syscall.Getpgrp())
		os.Exit(2)
	}
}

func setTestTerminalForegroundProcessGroup(t *testing.T, processGroupID int) {
	t.Helper()
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open controlling terminal: %v", err)
	}
	defer tty.Close()

	sigttouWasIgnored := signal.Ignored(syscall.SIGTTOU)
	if !sigttouWasIgnored {
		signal.Ignore(syscall.SIGTTOU)
		defer signal.Reset(syscall.SIGTTOU)
	}
	if err := setTerminalForegroundProcessGroup(tty, processGroupID); err != nil {
		t.Fatalf("set test terminal foreground process group: %v", err)
	}
}

func testTerminalForegroundProcessGroup(t *testing.T) int {
	t.Helper()
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open controlling terminal: %v", err)
	}
	defer tty.Close()

	processGroupID, err := terminalForegroundProcessGroup(tty)
	if err != nil {
		t.Fatalf("read terminal foreground process group: %v", err)
	}
	return processGroupID
}

func writeHelperPIDMarker(dir string, pid int) {
	path := filepath.Join(dir, strconv.Itoa(pid))
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write helper PID marker: %v\n", err)
		os.Exit(2)
	}
}

func processTreeHelperArgs(scenario string, args ...string) []string {
	result := []string{"-test.run=^TestProcessTreeHelper$", "--", scenario}
	return append(result, args...)
}

func processTreeHelperScenario(args []string) (string, []string, bool) {
	for i, arg := range args {
		if arg == "--" && i+1 < len(args) {
			return args[i+1], args[i+2:], true
		}
	}
	return "", nil, false
}

func waitForHelperPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(path)
		if err == nil {
			pid, err := strconv.Atoi(string(contents))
			if err != nil {
				t.Fatalf("parse helper pid %q: %v", contents, err)
			}
			return pid
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read helper pid: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("helper did not write pid to %s", path)
	return 0
}

func waitForHelperPIDs(t *testing.T, dir string, minimum int) []int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pids := helperPIDs(t, dir)
		if len(pids) >= minimum {
			return pids
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("helper wrote fewer than %d PID markers to %s", minimum, dir)
	return nil
}

func helperPIDs(t *testing.T, dir string) []int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read helper PID markers: %v", err)
	}
	pids := make([]int, 0, len(entries))
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			t.Fatalf("parse helper PID marker %q: %v", entry.Name(), err)
		}
		pids = append(pids, pid)
	}
	return pids
}

func waitForTestProcessExit(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("descendant process %d is still running", pid)
}

func waitForTestProcessGroupExit(t *testing.T, processGroupID int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-processGroupID, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("helper process group %d is still running", processGroupID)
}

func terminateTestProcess(pid int) {
	if pid > 0 {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}

func terminateTestProcessGroup(processGroupID int) {
	if processGroupID > 0 {
		_ = syscall.Kill(-processGroupID, syscall.SIGKILL)
	}
}

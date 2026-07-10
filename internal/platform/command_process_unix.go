//go:build darwin || linux

package platform

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const commandGroupRelayGracePeriod = 50 * time.Millisecond

const terminalSignalProxyScript = `
trap 'kill -INT "$1"' INT QUIT
trap 'kill -TSTP "$1"; kill -STOP -"$$"' TSTP TTIN TTOU
trap 'exit 0' TERM HUP
printf . >&3
exec 3>&-
while :; do
  sleep 3600 &
  wait $!
done
`

var terminalForegroundMu sync.Mutex

func configureCommandCancellation(cmd *exec.Cmd, waitDelay time.Duration, isolateProcessGroup bool) *commandCancellationState {
	state := &commandCancellationState{}
	cmd.WaitDelay = waitDelay

	if isolateProcessGroup {
		configureIsolatedProcessGroup(cmd, state)
	} else if !configureForegroundProcessGroup(cmd, state) {
		// Tests and non-interactive callers can explicitly select this path even
		// when /dev/tty is unavailable. A private process group still provides
		// deterministic ownership; only foreground terminal access is omitted.
		configureIsolatedProcessGroup(cmd, state)
	}

	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return state.capture(os.ErrProcessDone)
		}
		return state.capture(cancelCommandProcessGroup(state.ownedProcessGroupID(cmd.Process)))
	}
	return state
}

func configureIsolatedProcessGroup(cmd *exec.Cmd, state *commandCancellationState) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	state.finish = func() error {
		if cmd.Process == nil {
			return nil
		}
		return killCommandProcessGroup(state.ownedProcessGroupID(cmd.Process))
	}
}

func configureForegroundProcessGroup(cmd *exec.Cmd, state *commandCancellationState) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}

	terminalForegroundMu.Lock()
	parentProcessGroup := syscall.Getpgrp()
	proxy, err := startTerminalSignalProxy(os.Getpid())
	if err != nil {
		_ = tty.Close()
		terminalForegroundMu.Unlock()
		return false
	}
	processGroupID := proxy.Process.Pid
	state.processGroupID = processGroupID
	continueSignals := make(chan os.Signal, 1)
	stopContinueWatcher := make(chan struct{})
	continueWatcherDone := make(chan struct{})
	signal.Notify(continueSignals, syscall.SIGCONT)
	go func() {
		defer close(continueWatcherDone)
		for {
			select {
			case <-continueSignals:
				if err := resumeCommandProcessGroup(tty, parentProcessGroup, processGroupID); err != nil {
					state.capture(err)
					state.capture(killCommandProcessGroup(processGroupID))
				}
			case <-stopContinueWatcher:
				return
			}
		}
	}()

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    processGroupID,
	}
	state.afterStart = func() error {
		if err := setTerminalForegroundProcessGroup(tty, processGroupID); err != nil {
			return fmt.Errorf("set command foreground process group: %w", err)
		}
		if err := syscall.Kill(-processGroupID, syscall.SIGCONT); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("resume command foreground process group: %w", err)
		}
		return nil
	}
	state.finish = func() error {
		var finishErrors []error
		signal.Stop(continueSignals)
		close(stopContinueWatcher)
		<-continueWatcherDone
		if err := killCommandProcessGroup(processGroupID); err != nil {
			finishErrors = append(finishErrors, err)
		}
		_ = proxy.Process.Kill()
		_ = proxy.Wait()

		if err := restoreTerminalForegroundProcessGroup(tty, processGroupID, parentProcessGroup); err != nil {
			finishErrors = append(finishErrors, fmt.Errorf("restore terminal foreground process group: %w", err))
		}
		if err := tty.Close(); err != nil {
			finishErrors = append(finishErrors, fmt.Errorf("close controlling terminal: %w", err))
		}
		terminalForegroundMu.Unlock()
		return errors.Join(finishErrors...)
	}
	return true
}

func startTerminalSignalProxy(parentPID int) (*exec.Cmd, error) {
	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create terminal signal proxy readiness pipe: %w", err)
	}

	proxy := exec.Command("/bin/sh", "-c", terminalSignalProxyScript, "kitout-signal-proxy", strconv.Itoa(parentPID))
	proxy.ExtraFiles = []*os.File{readyWriter}
	proxy.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := proxy.Start(); err != nil {
		_ = readyReader.Close()
		_ = readyWriter.Close()
		return nil, fmt.Errorf("start terminal signal proxy: %w", err)
	}
	_ = readyWriter.Close()

	ready := []byte{0}
	_, readErr := io.ReadFull(readyReader, ready)
	_ = readyReader.Close()
	if readErr != nil {
		_ = proxy.Process.Kill()
		_ = proxy.Wait()
		return nil, fmt.Errorf("wait for terminal signal proxy: %w", readErr)
	}
	return proxy, nil
}

func cancelCommandProcessGroup(processGroupID int) error {
	relayErr := syscall.Kill(-processGroupID, syscall.SIGTERM)
	processDone := errors.Is(relayErr, syscall.ESRCH)
	if relayErr == nil {
		time.Sleep(commandGroupRelayGracePeriod)
	}

	groupErr := syscall.Kill(-processGroupID, syscall.SIGKILL)
	switch {
	case groupErr == nil:
		if relayErr != nil && !processDone {
			return fmt.Errorf("signal command process group for cancellation: %w", relayErr)
		}
		return nil
	case errors.Is(groupErr, syscall.ESRCH):
		if processDone {
			return os.ErrProcessDone
		}
		if relayErr != nil {
			return fmt.Errorf("signal command process group for cancellation: %w", relayErr)
		}
		return nil
	default:
		var relayFailure error
		if relayErr != nil && !processDone {
			relayFailure = fmt.Errorf("signal command process group for cancellation: %w", relayErr)
		}
		return errors.Join(relayFailure, fmt.Errorf("kill command process group %d: %w", processGroupID, groupErr))
	}
}

func killCommandProcessGroup(processGroupID int) error {
	if err := syscall.Kill(-processGroupID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("kill command process group %d: %w", processGroupID, err)
	}
	return nil
}

func resumeCommandProcessGroup(tty *os.File, parentProcessGroup, commandProcessGroup int) error {
	foregroundProcessGroup, err := terminalForegroundProcessGroup(tty)
	if err != nil {
		return fmt.Errorf("read terminal foreground process group after resume: %w", err)
	}
	if foregroundProcessGroup == parentProcessGroup {
		if err := setTerminalForegroundProcessGroup(tty, commandProcessGroup); err != nil {
			return fmt.Errorf("restore command foreground process group after resume: %w", err)
		}
	}
	if err := syscall.Kill(-commandProcessGroup, syscall.SIGCONT); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("resume command process group: %w", err)
	}
	return nil
}

func restoreTerminalForegroundProcessGroup(tty *os.File, commandProcessGroup, parentProcessGroup int) error {
	foregroundProcessGroup, err := terminalForegroundProcessGroup(tty)
	if err != nil {
		return fmt.Errorf("read terminal foreground process group: %w", err)
	}
	if foregroundProcessGroup != commandProcessGroup {
		return nil
	}

	sigttouWasIgnored := signal.Ignored(syscall.SIGTTOU)
	if !sigttouWasIgnored {
		signal.Ignore(syscall.SIGTTOU)
		defer signal.Reset(syscall.SIGTTOU)
	}
	return setTerminalForegroundProcessGroup(tty, parentProcessGroup)
}

func setTerminalForegroundProcessGroup(tty *os.File, processGroupID int) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		tty.Fd(),
		uintptr(syscall.TIOCSPGRP),
		uintptr(unsafe.Pointer(&processGroupID)),
	)
	runtime.KeepAlive(tty)
	runtime.KeepAlive(&processGroupID)
	if errno != 0 {
		return errno
	}
	return nil
}

func terminalForegroundProcessGroup(tty *os.File) (int, error) {
	var processGroupID int
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		tty.Fd(),
		uintptr(syscall.TIOCGPGRP),
		uintptr(unsafe.Pointer(&processGroupID)),
	)
	runtime.KeepAlive(tty)
	runtime.KeepAlive(&processGroupID)
	if errno != 0 {
		return 0, errno
	}
	return processGroupID, nil
}

func processHasControllingTerminal() bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	foregroundProcessGroup, foregroundErr := terminalForegroundProcessGroup(tty)
	_ = tty.Close()
	return foregroundErr == nil && foregroundProcessGroup == syscall.Getpgrp()
}

func commandProcessWasInterrupted(state *os.ProcessState) bool {
	if state == nil {
		return false
	}
	waitStatus, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return false
	}
	if waitStatus.Signaled() {
		return waitStatus.Signal() == syscall.SIGINT
	}
	return waitStatus.Exited() && waitStatus.ExitStatus() == 128+int(syscall.SIGINT)
}

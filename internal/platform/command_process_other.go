//go:build !darwin && !linux

package platform

import (
	"os"
	"os/exec"
	"time"
)

func configureCommandCancellation(cmd *exec.Cmd, waitDelay time.Duration, _ bool) *commandCancellationState {
	state := &commandCancellationState{}
	cmd.WaitDelay = waitDelay
	if cancel := cmd.Cancel; cancel != nil {
		cmd.Cancel = func() error {
			return state.capture(cancel())
		}
	}
	return state
}

func processHasControllingTerminal() bool {
	return false
}

func commandProcessWasInterrupted(_ *os.ProcessState) bool {
	return false
}

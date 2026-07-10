package cli

import (
	"context"
	"io"

	"github.com/vwall/kitout/internal/platform"
)

type runnerFactory struct {
	newRunner        func() platform.Runner
	newVerboseRunner func(stdout, stderr io.Writer) platform.Runner
}

type application struct {
	ctx            context.Context
	stdin          io.Reader
	interruptStdin func()
	stdout         io.Writer
	stderr         io.Writer
	runners        runnerFactory
}

func newApplication(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) application {
	return application{
		ctx:    ctx,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		runners: runnerFactory{
			newRunner: func() platform.Runner {
				return platform.WithTrustedCommandPath(platform.NewExecRunner())
			},
			newVerboseRunner: func(stdout, stderr io.Writer) platform.Runner {
				return platform.WithTrustedCommandPath(platform.NewVerboseExecRunner(stdout, stderr))
			},
		},
	}
}

func (app application) newRunner() platform.Runner {
	return app.runners.newRunner()
}

func (app application) newVerboseRunner() platform.Runner {
	return app.runners.newVerboseRunner(app.stdout, app.stderr)
}

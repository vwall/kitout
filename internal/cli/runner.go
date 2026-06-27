package cli

import (
	"io"

	"github.com/vwall/kitout/internal/platform"
)

var cliExecRunnerFactory = func() platform.Runner {
	return platform.WithTrustedCommandPath(platform.NewExecRunner())
}

var cliVerboseExecRunnerFactory = func(stdout, stderr io.Writer) platform.Runner {
	return platform.WithTrustedCommandPath(platform.NewVerboseExecRunner(stdout, stderr))
}

func newCLIExecRunner() platform.Runner {
	return cliExecRunnerFactory()
}

func newCLIVerboseExecRunner(stdout, stderr io.Writer) platform.Runner {
	return cliVerboseExecRunnerFactory(stdout, stderr)
}

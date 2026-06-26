package cli

import (
	"io"

	"github.com/vwall/kitout/internal/platform"
)

func newCLIExecRunner() platform.Runner {
	return platform.WithTrustedCommandPath(platform.NewExecRunner())
}

func newCLIVerboseExecRunner(stdout, stderr io.Writer) platform.Runner {
	return platform.WithTrustedCommandPath(platform.NewVerboseExecRunner(stdout, stderr))
}

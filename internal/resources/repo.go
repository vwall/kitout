package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const repoType = "repo"

// RepoResource ensures a Git repository exists at a path with the expected origin.
type RepoResource struct {
	path   string
	url    string
	branch string
	runner platform.Runner
}

var _ engine.Resource = RepoResource{}

// NewRepo returns a resource for one Git checkout.
func NewRepo(path, url, branch string, runner platform.Runner) RepoResource {
	return RepoResource{path: path, url: url, branch: branch, runner: runner}
}

func (resource RepoResource) ID() string {
	return repoType + ":" + resource.path
}

func (resource RepoResource) Type() string {
	return repoType
}

func (resource RepoResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	info, err := os.Stat(resource.path)
	if errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateMissing, "repository is missing"), nil
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect repository path"), err
	}
	if !info.IsDir() {
		return resource.status(engine.StateChanged, "path exists but is not a directory"), nil
	}

	result, err := resource.runner.Run(ctx, "git", "-C", resource.path, "rev-parse", "--show-toplevel")
	if err != nil || result.Stdout == "" {
		return resource.status(engine.StateChanged, "path exists but is not a Git repository"), nil
	}
	rootInfo, err := os.Stat(strings.TrimSuffix(result.Stdout, "\n"))
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect repository root"), err
	}
	if !os.SameFile(info, rootInfo) {
		return resource.status(engine.StateChanged, "path is inside a Git repository but is not its root"), nil
	}

	result, err = resource.runner.Run(ctx, "git", "-C", resource.path, "remote", "get-url", "origin")
	if err != nil {
		return resource.status(engine.StateChanged, "repository is missing origin remote"), nil
	}
	if strings.TrimSpace(result.Stdout) != resource.url {
		return resource.status(engine.StateChanged, "repository origin does not match config"), nil
	}

	return resource.status(engine.StateSatisfied, "repository exists"), nil
}

func (resource RepoResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "repository already exists"), nil
	case engine.StateMissing:
		args := []string{"clone"}
		if resource.branch != "" {
			args = append(args, "--branch", resource.branch)
		}
		args = append(args, resource.url, resource.path)
		if _, err := resource.runner.Run(ctx, "git", args...); err != nil {
			return resource.applyResult("clone", false, "could not clone repository"), err
		}
		return resource.applyResult("clone", true, "cloned repository"), nil
	case engine.StateChanged:
		err := fmt.Errorf("cannot clone repository into %s: path already exists with different contents", resource.path)
		return resource.applyResult("fail", false, err.Error()), err
	default:
		err := fmt.Errorf("cannot apply repository %s from state %s", resource.path, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource RepoResource) validate() error {
	if resource.path == "" {
		return errors.New("repository path is required")
	}
	if resource.url == "" {
		return errors.New("repository URL is required")
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	return nil
}

func (resource RepoResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource RepoResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource RepoResource) details() map[string]string {
	details := map[string]string{
		"path": resource.path,
		"url":  resource.url,
	}
	if resource.branch != "" {
		details["branch"] = resource.branch
	}
	return details
}

package resources

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vwall/kitout/internal/engine"
)

const directoryType = "directory"

// DirectoryResource ensures a filesystem path exists as a directory.
type DirectoryResource struct {
	path string
}

var _ engine.Resource = DirectoryResource{}

// NewDirectory returns a resource for one desired directory path.
func NewDirectory(path string) DirectoryResource {
	return DirectoryResource{path: path}
}

// ID returns the stable resource identifier.
func (resource DirectoryResource) ID() string {
	return directoryType + ":" + resource.path
}

// Type returns the resource type.
func (resource DirectoryResource) Type() string {
	return directoryType
}

// Status checks whether the path exists and is a directory.
func (resource DirectoryResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return resource.status(engine.StateFailed, "context canceled while checking directory"), err
	}
	if resource.path == "" {
		err := errors.New("directory path is required")
		return resource.status(engine.StateFailed, err.Error()), err
	}

	info, err := os.Stat(resource.path)
	if errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateMissing, "directory is missing"), nil
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect directory"), err
	}
	if !info.IsDir() {
		return resource.status(engine.StateChanged, "path exists but is not a directory"), nil
	}

	return resource.status(engine.StateSatisfied, "directory exists"), nil
}

// Apply creates the directory and any missing parent directories.
func (resource DirectoryResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "directory already exists"), nil
	case engine.StateMissing:
		if err := os.MkdirAll(resource.path, 0o755); err != nil {
			return resource.applyResult("create", false, "could not create directory"), err
		}
		return resource.applyResult("create", true, "created directory"), nil
	case engine.StateChanged:
		err := fmt.Errorf("cannot create directory %s: path exists and is not a directory", resource.path)
		return resource.applyResult("fail", false, err.Error()), err
	default:
		err := fmt.Errorf("cannot apply directory %s from state %s", resource.path, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource DirectoryResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return engine.StatusResult{
		ResourceID: resource.ID(),
		Type:       resource.Type(),
		State:      state,
		Message:    message,
		Details: map[string]string{
			"path": resource.path,
		},
	}
}

func (resource DirectoryResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return engine.ApplyResult{
		ResourceID: resource.ID(),
		Type:       resource.Type(),
		Action:     action,
		Changed:    changed,
		Message:    message,
		Details: map[string]string{
			"path": resource.path,
		},
	}
}

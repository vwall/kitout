package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vwall/kitout/internal/engine"
)

const copyType = "copy"

// CopyResource ensures a target file or directory contains a physical copy of a source.
type CopyResource struct {
	source  string
	target  string
	replace bool
}

var _ engine.Resource = CopyResource{}

// NewCopy returns a resource for one desired file or directory copy.
func NewCopy(source, target string, replace bool) CopyResource {
	return CopyResource{source: source, target: target, replace: replace}
}

func (resource CopyResource) ID() string {
	return copyType + ":" + resource.target
}

func (resource CopyResource) Type() string {
	return copyType
}

func (resource CopyResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return resource.status(engine.StateFailed, "context canceled while checking copy"), err
	}
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	sourceInfo, err := os.Lstat(resource.source)
	if errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateFailed, "copy source is missing"), nil
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect copy source"), err
	}
	if err := validateCopySourceTree(resource.source, sourceInfo); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if err := validateCopyTargetAncestors(resource.target); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	targetInfo, err := os.Lstat(resource.target)
	if errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateMissing, "copy target is missing"), nil
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect copy target"), err
	}

	matches, err := copyTargetsMatch(resource.source, resource.target, sourceInfo, targetInfo)
	if err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if matches {
		return resource.status(engine.StateSatisfied, "copy target matches source"), nil
	}
	if resource.replace {
		return resource.status(engine.StateChanged, "copy target differs from source"), nil
	}

	return resource.status(engine.StateFailed, "copy target differs and replacement is not allowed"), nil
}

func (resource CopyResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "copy target already matches source"), nil
	case engine.StateMissing:
		if err := copyPath(resource.source, resource.target); err != nil {
			return resource.applyResult("create", false, "could not copy source to target"), err
		}
		return resource.applyResult("create", true, "copied source to target"), nil
	case engine.StateChanged:
		if !resource.replace {
			err := fmt.Errorf("cannot replace copy target %s: replacement is not allowed", resource.target)
			return resource.applyResult("fail", false, err.Error()), err
		}
		if err := os.RemoveAll(resource.target); err != nil {
			return resource.applyResult("replace", false, "could not remove existing target"), err
		}
		if err := copyPath(resource.source, resource.target); err != nil {
			return resource.applyResult("replace", false, "could not copy replacement target"), err
		}
		return resource.applyResult("replace", true, "replaced copy target"), nil
	case engine.StateFailed:
		err := errors.New(status.Message)
		return resource.applyResult("fail", false, status.Message), err
	default:
		err := fmt.Errorf("cannot apply copy %s from state %s", resource.target, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource CopyResource) validate() error {
	if resource.source == "" {
		return errors.New("copy source is required")
	}
	if resource.target == "" {
		return errors.New("copy target is required")
	}
	return nil
}

func validateCopySource(path string, info fs.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("copy source %s must not be a symlink", path)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return fmt.Errorf("copy source %s must be a file or directory", path)
	}
	return nil
}

func validateCopySourceTree(path string, info fs.FileInfo) error {
	if err := validateCopySource(path, info); err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(path, func(childPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		childInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if childPath == path {
			return nil
		}
		if childInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy source contains symlink %s", childPath)
		}
		if !childInfo.IsDir() && !childInfo.Mode().IsRegular() {
			return fmt.Errorf("copy source contains unsupported path %s", childPath)
		}
		return nil
	})
}

func validateCopyTargetAncestors(path string) error {
	for _, ancestor := range pathAncestors(path) {
		info, err := os.Lstat(ancestor)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("could not inspect copy target ancestor %s: %w", ancestor, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if isAllowedDarwinSystemSymlink(ancestor) {
				continue
			}
			return fmt.Errorf("copy target ancestor %s must not be a symlink", ancestor)
		}
		if !info.IsDir() {
			return fmt.Errorf("copy target ancestor %s must be a directory", ancestor)
		}
	}
	return nil
}

func pathAncestors(path string) []string {
	parent := filepath.Dir(filepath.Clean(path))
	var reversed []string
	for parent != "." && parent != "" && parent != string(filepath.Separator) {
		reversed = append(reversed, parent)
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parent = next
	}

	ancestors := make([]string, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		ancestors = append(ancestors, reversed[i])
	}
	return ancestors
}

func isAllowedDarwinSystemSymlink(path string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	expectedTargets := map[string]string{
		"/etc": "/private/etc",
		"/tmp": "/private/tmp",
		"/var": "/private/var",
	}
	expected, ok := expectedTargets[filepath.Clean(path)]
	if !ok {
		return false
	}

	linkTarget, err := os.Readlink(path)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(path), linkTarget)
	}
	return filepath.Clean(linkTarget) == expected
}

func copyTargetsMatch(source, target string, sourceInfo, targetInfo fs.FileInfo) (bool, error) {
	switch {
	case sourceInfo.IsDir():
		if !targetInfo.IsDir() || targetInfo.Mode()&os.ModeSymlink != 0 {
			return false, nil
		}
		return directoriesMatch(source, target)
	case sourceInfo.Mode().IsRegular():
		if !targetInfo.Mode().IsRegular() || targetInfo.Mode()&os.ModeSymlink != 0 {
			return false, nil
		}
		return filesMatch(source, target)
	default:
		return false, fmt.Errorf("copy source %s must be a file or directory", source)
	}
}

func filesMatch(source, target string) (bool, error) {
	sourceContents, err := os.ReadFile(source)
	if err != nil {
		return false, fmt.Errorf("could not read copy source %s: %w", source, err)
	}
	targetContents, err := os.ReadFile(target)
	if err != nil {
		return false, fmt.Errorf("could not read copy target %s: %w", target, err)
	}
	return bytes.Equal(sourceContents, targetContents), nil
}

func directoriesMatch(source, target string) (bool, error) {
	sourceEntries := make(map[string]struct{})
	matches := true
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		sourceEntries[rel] = struct{}{}

		sourceInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if sourceInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy source contains symlink %s", path)
		}
		if !sourceInfo.IsDir() && !sourceInfo.Mode().IsRegular() {
			return fmt.Errorf("copy source contains unsupported path %s", path)
		}

		targetPath := filepath.Join(target, rel)
		targetInfo, err := os.Lstat(targetPath)
		if errors.Is(err, os.ErrNotExist) {
			matches = false
			return nil
		}
		if err != nil {
			return err
		}
		entryMatches, err := copyTargetsMatch(path, targetPath, sourceInfo, targetInfo)
		if err != nil {
			return err
		}
		if !entryMatches {
			matches = false
		}
		return nil
	})
	if err != nil || !matches {
		return matches, err
	}

	err = filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(target, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if _, ok := sourceEntries[rel]; !ok {
			matches = false
		}
		return nil
	})
	return matches, err
}

func copyPath(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if err := validateCopySourceTree(source, info); err != nil {
		return err
	}
	if err := validateCopyTargetAncestors(target); err != nil {
		return err
	}
	if info.IsDir() {
		return copyDirectory(source, target)
	}
	return copyFile(source, target, info.Mode().Perm())
}

func copyDirectory(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		targetPath := target
		if rel != "." {
			targetPath = filepath.Join(target, rel)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy source contains symlink %s", path)
		}
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}
		if info.Mode().IsRegular() {
			return copyFile(path, targetPath, info.Mode().Perm())
		}
		return fmt.Errorf("copy source contains unsupported path %s", path)
	})
}

func copyFile(source, target string, mode fs.FileMode) error {
	contents, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, contents, mode)
}

func (resource CopyResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource CopyResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource CopyResource) details() map[string]string {
	replace := "false"
	if resource.replace {
		replace = "true"
	}
	return map[string]string{
		"source":  resource.source,
		"target":  resource.target,
		"replace": replace,
	}
}

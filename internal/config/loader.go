package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the default Kitout config file location.
const DefaultPath = "~/.config/kitout/kitout.yaml"

// LocalPath is the repo-local Kitout config file location.
const LocalPath = "./kitout.yaml"

// LoadedConfig is a config loaded from disk with its resolved source path.
type LoadedConfig struct {
	Path     string
	Config   Config
	Warnings []ConfigWarning
}

// AmbiguousConfigError reports that Kitout found both implicit config paths and
// needs the caller to choose one explicitly.
type AmbiguousConfigError struct {
	LocalPath string
	HomePath  string
}

func (err AmbiguousConfigError) Error() string {
	return fmt.Sprintf("both local config %s and home config %s exist; pass --config to choose one", err.LocalPath, err.HomePath)
}

// ParseError wraps YAML parsing and schema decode failures for a config file.
type ParseError struct {
	Path string
	Err  error
}

func (err ParseError) Error() string {
	return fmt.Sprintf("parse config %s: %v", err.Path, err.Err)
}

func (err ParseError) Unwrap() error {
	return err.Err
}

// ResolvePath expands a user-provided config path into a clean filesystem path.
func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}

	expanded := os.ExpandEnv(path)
	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}

	return filepath.Clean(expanded), nil
}

// SelectPath resolves the config path Kitout should use when loading config.
//
// An explicit path always wins. Without one, Kitout can use an unambiguous
// local or home config. If both exist, the caller must pass --config.
func SelectPath(explicitPath string) (string, error) {
	if explicitPath != "" {
		return ResolvePath(explicitPath)
	}

	localPath, err := ResolvePath(LocalPath)
	if err != nil {
		return "", err
	}
	localAbs, err := filepath.Abs(localPath)
	if err != nil {
		return "", err
	}
	localExists := false
	if _, err := os.Stat(localPath); err == nil {
		localExists = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return localAbs, nil
	}

	homePath, err := ResolvePath(DefaultPath)
	if err != nil {
		return "", err
	}
	homeExists := false
	if _, err := os.Stat(homePath); err == nil {
		homeExists = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return homePath, nil
	}

	if localExists && homeExists {
		return "", AmbiguousConfigError{LocalPath: localAbs, HomePath: homePath}
	}
	if localExists {
		return localAbs, nil
	}

	return homePath, nil
}

// LoadFile reads and decodes a Kitout YAML config file.
//
// Public config path behavior is intentionally explicit: path-bearing resource
// fields are normalized in the returned Config. Relative resource paths resolve
// from the directory containing this config file, while absolute paths, "~", and
// environment variables keep their usual meaning.
func LoadFile(path string) (LoadedConfig, error) {
	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return LoadedConfig{}, err
	}

	contents, err := os.ReadFile(resolvedPath)
	if err != nil {
		return LoadedConfig{}, fmt.Errorf("read config %s: %w", resolvedPath, err)
	}

	topLevelCasksSet, err := hasRootYAMLField(contents, "casks")
	if err != nil {
		return LoadedConfig{}, ParseError{Path: resolvedPath, Err: err}
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return LoadedConfig{}, ParseError{Path: resolvedPath, Err: err}
	}
	cfg.topLevelCasksSet = topLevelCasksSet

	if err := validateDecodedConfig(cfg); err != nil {
		return LoadedConfig{}, err
	}
	warnings := Warnings(cfg)

	cfg = resolveResourcePaths(filepath.Dir(resolvedPath), cfg)
	if err := Validate(cfg); err != nil {
		return LoadedConfig{}, err
	}

	return LoadedConfig{
		Path:     resolvedPath,
		Config:   cfg,
		Warnings: warnings,
	}, nil
}

func hasRootYAMLField(contents []byte, field string) (bool, error) {
	var document yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(contents)).Decode(&document); err != nil {
		return false, err
	}

	if document.Kind == yaml.DocumentNode {
		if len(document.Content) == 0 {
			return false, nil
		}
		document = *document.Content[0]
	}
	if document.Kind != yaml.MappingNode {
		return false, nil
	}

	for i := 0; i+1 < len(document.Content); i += 2 {
		if document.Content[i].Value == field {
			return true, nil
		}
	}
	return false, nil
}

func resolveResourcePaths(baseDir string, cfg Config) Config {
	for i, path := range cfg.Directories {
		cfg.Directories[i] = resolveResourcePath(baseDir, path)
	}

	for i, item := range cfg.ASDF.ToolVersions {
		cfg.ASDF.ToolVersions[i].Path = resolveResourcePath(baseDir, item.Path)
	}

	for i, repo := range cfg.Repos {
		cfg.Repos[i].Path = resolveResourcePath(baseDir, repo.Path)
	}

	for i, copy := range cfg.Copies {
		cfg.Copies[i].Source = resolveResourcePath(baseDir, copy.Source)
		cfg.Copies[i].Target = resolveResourcePath(baseDir, copy.Target)
	}

	for i, symlink := range cfg.Symlinks {
		cfg.Symlinks[i].Source = resolveResourcePath(baseDir, symlink.Source)
		cfg.Symlinks[i].Target = resolveResourcePath(baseDir, symlink.Target)
	}

	for i, group := range cfg.SymlinkGroups {
		cfg.SymlinkGroups[i].SourceRoot = resolveResourcePath(baseDir, group.SourceRoot)
		cfg.SymlinkGroups[i].TargetRoot = resolveResourcePath(baseDir, group.TargetRoot)
		for j, path := range group.Paths {
			cfg.SymlinkGroups[i].Paths[j] = filepath.Clean(path)
		}
	}

	for i, key := range cfg.SSH.Keys {
		cfg.SSH.Keys[i].Path = resolveResourcePath(baseDir, key.Path)
	}

	return cfg
}

func resolveResourcePath(baseDir, path string) string {
	expanded := os.ExpandEnv(path)
	if expanded == "" {
		return ""
	}

	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if expanded == "~" {
				expanded = home
			} else {
				expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
			}
		}
	}

	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded)
	}

	return filepath.Clean(filepath.Join(baseDir, expanded))
}

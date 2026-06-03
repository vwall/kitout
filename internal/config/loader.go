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

// LoadedConfig is a config loaded from disk with its resolved source path.
type LoadedConfig struct {
	Path   string
	Config Config
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

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return LoadedConfig{}, ParseError{Path: resolvedPath, Err: err}
	}

	if err := Validate(cfg); err != nil {
		return LoadedConfig{}, err
	}

	cfg = resolveResourcePaths(filepath.Dir(resolvedPath), cfg)

	return LoadedConfig{
		Path:   resolvedPath,
		Config: cfg,
	}, nil
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

	for i, symlink := range cfg.Symlinks {
		cfg.Symlinks[i].Source = resolveResourcePath(baseDir, symlink.Source)
		cfg.Symlinks[i].Target = resolveResourcePath(baseDir, symlink.Target)
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

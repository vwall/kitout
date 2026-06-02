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
		return LoadedConfig{}, fmt.Errorf("parse config %s: %w", resolvedPath, err)
	}

	return LoadedConfig{
		Path:   resolvedPath,
		Config: cfg,
	}, nil
}

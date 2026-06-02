package config

import (
	"fmt"
	"strings"
)

// ValidationError describes one specific config schema problem.
type ValidationError struct {
	Field   string
	Message string
}

func (err ValidationError) Error() string {
	if err.Field == "" {
		return err.Message
	}

	return err.Field + " " + err.Message
}

// ValidationErrors is a structured collection of config validation failures.
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	switch len(errs) {
	case 0:
		return "Invalid config"
	case 1:
		return "Invalid config: " + errs[0].Error()
	default:
		messages := make([]string, 0, len(errs))
		for _, err := range errs {
			messages = append(messages, err.Error())
		}
		return "Invalid config: " + strings.Join(messages, "; ")
	}
}

// Validate checks the decoded config against Kitout's documented schema.
func Validate(cfg Config) error {
	var errs ValidationErrors

	if cfg.Version == 0 {
		errs.add("version", "is required")
	} else if cfg.Version != CurrentVersion {
		errs.add("version", fmt.Sprintf("must be %d", CurrentVersion))
	}

	errs.requireStrings("brew.packages", cfg.Brew.Packages)
	errs.requireStrings("casks", cfg.Casks)
	errs.requireStrings("directories", cfg.Directories)

	errs.detectDuplicateStrings("brew.packages", cfg.Brew.Packages)
	errs.detectDuplicateStrings("casks", cfg.Casks)
	errs.detectDuplicateStrings("directories", cfg.Directories)

	for i, repo := range cfg.Repos {
		errs.requireString(fmt.Sprintf("repos[%d].path", i), repo.Path)
		errs.requireString(fmt.Sprintf("repos[%d].url", i), repo.URL)
	}
	errs.detectDuplicates(repoPathKeys(cfg.Repos))

	for i, symlink := range cfg.Symlinks {
		errs.requireString(fmt.Sprintf("symlinks[%d].source", i), symlink.Source)
		errs.requireString(fmt.Sprintf("symlinks[%d].target", i), symlink.Target)
	}
	errs.detectDuplicates(symlinkTargetKeys(cfg.Symlinks))

	for i, item := range cfg.MacOSDefaults {
		errs.requireString(fmt.Sprintf("macos_defaults[%d].domain", i), item.Domain)
		errs.requireString(fmt.Sprintf("macos_defaults[%d].key", i), item.Key)
		errs.requireString(fmt.Sprintf("macos_defaults[%d].type", i), item.Type)
		if item.Value == nil {
			errs.add(fmt.Sprintf("macos_defaults[%d].value", i), "is required")
		}
		if item.Type != "" && !validMacOSDefaultType(item.Type) {
			errs.add(fmt.Sprintf("macos_defaults[%d].type", i), "must be one of bool, int, float, string")
		}
	}
	errs.detectDuplicates(macOSDefaultKeys(cfg.MacOSDefaults))

	for i, command := range cfg.Shell {
		errs.requireString(fmt.Sprintf("shell[%d].name", i), command.Name)
		errs.requireString(fmt.Sprintf("shell[%d].command", i), command.Command)
	}
	errs.detectDuplicates(shellNameKeys(cfg.Shell))

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (errs *ValidationErrors) add(field, message string) {
	*errs = append(*errs, ValidationError{Field: field, Message: message})
}

func (errs *ValidationErrors) requireString(field, value string) {
	if strings.TrimSpace(value) == "" {
		errs.add(field, "is required")
	}
}

func (errs *ValidationErrors) requireStrings(field string, values []string) {
	for i, value := range values {
		errs.requireString(fmt.Sprintf("%s[%d]", field, i), value)
	}
}

func (errs *ValidationErrors) detectDuplicateStrings(field string, values []string) {
	keys := make([]duplicateKey, 0, len(values))
	for i, value := range values {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("%s[%d]", field, i),
			Value: value,
		})
	}
	errs.detectDuplicates(keys)
}

func (errs *ValidationErrors) detectDuplicates(keys []duplicateKey) {
	firstFields := make(map[string]string)
	for _, key := range keys {
		value := strings.TrimSpace(key.Value)
		if value == "" {
			continue
		}

		if firstField, ok := firstFields[value]; ok {
			errs.add(key.Field, fmt.Sprintf("duplicates %s (%s)", firstField, key.display()))
			continue
		}

		firstFields[value] = key.Field
	}
}

type duplicateKey struct {
	Field   string
	Value   string
	Display string
}

func (key duplicateKey) display() string {
	if key.Display != "" {
		return key.Display
	}

	return key.Value
}

func repoPathKeys(repos []Repo) []duplicateKey {
	keys := make([]duplicateKey, 0, len(repos))
	for i, repo := range repos {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("repos[%d].path", i),
			Value: repo.Path,
		})
	}
	return keys
}

func symlinkTargetKeys(symlinks []Symlink) []duplicateKey {
	keys := make([]duplicateKey, 0, len(symlinks))
	for i, symlink := range symlinks {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("symlinks[%d].target", i),
			Value: symlink.Target,
		})
	}
	return keys
}

func macOSDefaultKeys(items []MacOSDefault) []duplicateKey {
	keys := make([]duplicateKey, 0, len(items))
	for i, item := range items {
		value := ""
		display := ""
		if strings.TrimSpace(item.Domain) != "" && strings.TrimSpace(item.Key) != "" {
			value = item.Domain + "\x00" + item.Key
			display = item.Domain + "/" + item.Key
		}
		keys = append(keys, duplicateKey{
			Field:   fmt.Sprintf("macos_defaults[%d]", i),
			Value:   value,
			Display: display,
		})
	}
	return keys
}

func shellNameKeys(commands []ShellCommand) []duplicateKey {
	keys := make([]duplicateKey, 0, len(commands))
	for i, command := range commands {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("shell[%d].name", i),
			Value: command.Name,
		})
	}
	return keys
}

func validMacOSDefaultType(value string) bool {
	switch value {
	case "bool", "int", "float", "string":
		return true
	default:
		return false
	}
}

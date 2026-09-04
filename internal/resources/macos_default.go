package resources

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const macOSDefaultType = "macos_default"

// MacOSDefaultResource ensures one macOS defaults key has a configured value.
type MacOSDefaultResource struct {
	domain    string
	key       string
	valueType string
	value     any
	runner    platform.Runner
}

var _ engine.Resource = MacOSDefaultResource{}

// NewMacOSDefault returns a resource for one macOS defaults value.
func NewMacOSDefault(domain, key, valueType string, value any, runner platform.Runner) MacOSDefaultResource {
	return MacOSDefaultResource{
		domain:    domain,
		key:       key,
		valueType: valueType,
		value:     value,
		runner:    runner,
	}
}

func (resource MacOSDefaultResource) ID() string {
	return macOSDefaultType + ":" + resource.domain + "/" + resource.key
}

func (resource MacOSDefaultResource) Type() string {
	return macOSDefaultType
}

func (resource MacOSDefaultResource) Status(ctx context.Context) (engine.StatusResult, error) {
	desired, err := resource.desiredValue()
	if err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	result, err := resource.runner.Run(ctx, "defaults", "read", resource.domain, resource.key)
	if err == nil {
		if desired.matches(result.Stdout) {
			typeResult, typeErr := resource.runner.Run(ctx, "defaults", "read-type", resource.domain, resource.key)
			if typeErr != nil {
				return resource.status(engine.StateFailed, "could not inspect default type"), typeErr
			}
			if !desired.matchesType(typeResult.Stdout) {
				return resource.status(engine.StateChanged, "default type differs"), nil
			}
			return resource.status(engine.StateSatisfied, "default is set"), nil
		}
		return resource.status(engine.StateChanged, "default value differs"), nil
	}
	if isExitCode(err, 1) {
		return resource.status(engine.StateMissing, "default is missing"), nil
	}

	return resource.status(engine.StateFailed, "could not inspect default"), err
}

func (resource MacOSDefaultResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	desired, err := resource.desiredValue()
	if err != nil {
		return resource.applyResult("fail", false, err.Error()), err
	}

	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "default already set"), nil
	case engine.StateMissing, engine.StateChanged:
		args := []string{"write", resource.domain, resource.key, desired.writeFlag(), desired.writeValue}
		if _, err := platform.WithBoundedOutput(resource.runner).Run(ctx, "defaults", args...); err != nil {
			return resource.applyResult("write", false, "could not write default"), err
		}
		return resource.applyResult("write", true, "wrote default"), nil
	default:
		err := fmt.Errorf("cannot apply macOS default %s/%s from state %s", resource.domain, resource.key, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource MacOSDefaultResource) desiredValue() (macOSDefaultValue, error) {
	if resource.domain == "" {
		return macOSDefaultValue{}, errors.New("macOS default domain is required")
	}
	if resource.key == "" {
		return macOSDefaultValue{}, errors.New("macOS default key is required")
	}
	if resource.valueType == "" {
		return macOSDefaultValue{}, errors.New("macOS default type is required")
	}
	if resource.runner == nil {
		return macOSDefaultValue{}, errors.New("command runner is required")
	}

	return newMacOSDefaultValue(resource.valueType, resource.value)
}

func (resource MacOSDefaultResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource MacOSDefaultResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource MacOSDefaultResource) details() map[string]string {
	details := map[string]string{
		"domain": resource.domain,
		"key":    resource.key,
		"type":   resource.valueType,
	}
	if desired, err := newMacOSDefaultValue(resource.valueType, resource.value); err == nil {
		details["value"] = desired.writeValue
	}
	return details
}

type macOSDefaultValue struct {
	typ        string
	writeValue string
}

func newMacOSDefaultValue(valueType string, value any) (macOSDefaultValue, error) {
	switch valueType {
	case "bool":
		boolValue, ok := value.(bool)
		if !ok {
			return macOSDefaultValue{}, errors.New("macOS default bool value must be true or false")
		}
		if boolValue {
			return macOSDefaultValue{typ: valueType, writeValue: "true"}, nil
		}
		return macOSDefaultValue{typ: valueType, writeValue: "false"}, nil
	case "int":
		intValue, ok := int64Value(value)
		if !ok {
			return macOSDefaultValue{}, errors.New("macOS default int value must be an integer")
		}
		return macOSDefaultValue{typ: valueType, writeValue: strconv.FormatInt(intValue, 10)}, nil
	case "float":
		floatValue, ok := float64Value(value)
		if !ok || math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return macOSDefaultValue{}, errors.New("macOS default float value must be a finite number")
		}
		return macOSDefaultValue{typ: valueType, writeValue: strconv.FormatFloat(floatValue, 'f', -1, 64)}, nil
	case "string":
		stringValue, ok := value.(string)
		if !ok {
			return macOSDefaultValue{}, errors.New("macOS default string value must be a string")
		}
		return macOSDefaultValue{typ: valueType, writeValue: stringValue}, nil
	default:
		return macOSDefaultValue{}, fmt.Errorf("unsupported macOS default type %q", valueType)
	}
}

func (value macOSDefaultValue) writeFlag() string {
	return "-" + value.typ
}

func (value macOSDefaultValue) matches(stdout string) bool {
	if value.typ == "string" {
		// defaults read adds one newline; whitespace in the value is significant.
		return strings.TrimSuffix(stdout, "\n") == value.writeValue
	}
	actual := strings.TrimSpace(stdout)
	switch value.typ {
	case "bool":
		actualBool, ok := parseDefaultBool(actual)
		return ok && actualBool == (value.writeValue == "true")
	case "int":
		actualInt, err := strconv.ParseInt(actual, 10, 64)
		if err != nil {
			return false
		}
		expectedInt, err := strconv.ParseInt(value.writeValue, 10, 64)
		return err == nil && actualInt == expectedInt
	case "float":
		actualFloat, err := strconv.ParseFloat(actual, 64)
		if err != nil {
			return false
		}
		expectedFloat, err := strconv.ParseFloat(value.writeValue, 64)
		return err == nil && actualFloat == expectedFloat
	default:
		return false
	}
}

func (value macOSDefaultValue) matchesType(stdout string) bool {
	typ := value.typ
	switch typ {
	case "bool":
		typ = "boolean"
	case "int":
		typ = "integer"
	}
	return strings.TrimSpace(stdout) == "Type is "+typ
}

func parseDefaultBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes":
		return true, true
	case "0", "false", "no":
		return false, true
	default:
		return false, false
	}
}

func int64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func float64Value(value any) (float64, bool) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

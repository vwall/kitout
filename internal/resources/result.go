package resources

import "github.com/vwall/kitout/internal/engine"

func statusResult(id, typ string, state engine.ResourceState, message string, details map[string]string) engine.StatusResult {
	return engine.StatusResult{
		ResourceID: id,
		Type:       typ,
		State:      state,
		Message:    message,
		Details:    details,
	}
}

func applyResult(id, typ, action string, changed bool, message string, details map[string]string) engine.ApplyResult {
	return engine.ApplyResult{
		ResourceID: id,
		Type:       typ,
		Action:     action,
		Changed:    changed,
		Message:    message,
		Details:    details,
	}
}

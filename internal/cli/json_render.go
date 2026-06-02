package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
)

type jsonRenderer struct {
	stdout io.Writer
}

type jsonStatusResponse struct {
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Config  *jsonConfigStatus `json:"config,omitempty"`
	Status  *jsonStatusState  `json:"status,omitempty"`
	Error   *jsonError        `json:"error,omitempty"`
}

type jsonConfigStatus struct {
	Path  string `json:"path,omitempty"`
	Valid bool   `json:"valid"`
}

type jsonStatusState struct {
	Implemented bool   `json:"implemented"`
	Message     string `json:"message"`
}

type jsonError struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Details []jsonErrorDetail `json:"details,omitempty"`
}

type jsonErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func newJSONRenderer(stdout io.Writer) jsonRenderer {
	return jsonRenderer{stdout: stdout}
}

func (r jsonRenderer) renderStatusNotImplemented(path string) error {
	return r.write(jsonStatusResponse{
		Command: "status",
		OK:      true,
		Config: &jsonConfigStatus{
			Path:  path,
			Valid: true,
		},
		Status: &jsonStatusState{
			Implemented: false,
			Message:     "Status checks are not implemented yet.",
		},
	})
}

func (r jsonRenderer) renderValidationErrors(errs config.ValidationErrors) error {
	details := make([]jsonErrorDetail, 0, len(errs))
	for _, err := range errs {
		details = append(details, jsonErrorDetail{
			Field:   err.Field,
			Message: err.Message,
		})
	}

	return r.write(jsonStatusResponse{
		Command: "status",
		OK:      false,
		Config: &jsonConfigStatus{
			Valid: false,
		},
		Error: &jsonError{
			Type:    "validation",
			Message: errs.Error(),
			Details: details,
		},
	})
}

func (r jsonRenderer) renderParseError(err config.ParseError) error {
	return r.write(jsonStatusResponse{
		Command: "status",
		OK:      false,
		Config: &jsonConfigStatus{
			Path:  err.Path,
			Valid: false,
		},
		Error: &jsonError{
			Type:    "parse",
			Message: fmt.Sprintf("Invalid config: %v", err),
		},
	})
}

func (r jsonRenderer) renderConfigLoadFailure(err error) error {
	return r.write(jsonStatusResponse{
		Command: "status",
		OK:      false,
		Error: &jsonError{
			Type:    "runtime",
			Message: fmt.Sprintf("Failed to load config: %v", err),
		},
	})
}

func (r jsonRenderer) write(response jsonStatusResponse) error {
	encoder := json.NewEncoder(r.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}

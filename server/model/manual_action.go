package model

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ManualActionField describes a single input value that can be collected from the UI.
type ManualActionField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     string `json:"default,omitempty"`
	Type        string `json:"type,omitempty"`
}

// ManualActionRequestDefinition contains the request meta data that is required to
// trigger a manual action webhook.
type ManualActionRequestDefinition struct {
	URL      string              `json:"url"`
	Method   string              `json:"method"`
	BodyType string              `json:"bodyType,omitempty"`
	Query    []ManualActionField `json:"query,omitempty"`
	Headers  []ManualActionField `json:"headers,omitempty"`
	Body     []ManualActionField `json:"body,omitempty"`
	Timeout  string              `json:"timeout,omitempty"`
}

// ManualActionDefinition describes a triggerable action that can be initiated from the UI.
type ManualActionDefinition struct {
	ID          string                        `json:"id"`
	Title       string                        `json:"title"`
	Description string                        `json:"description,omitempty"`
	Request     ManualActionRequestDefinition `json:"request"`
}

// Validate verifies that the manual action definition is usable.
func (m ManualActionDefinition) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("manual action definition is missing id")
	}
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("manual action definition %s is missing title", m.ID)
	}
	if strings.TrimSpace(m.Request.URL) == "" {
		return fmt.Errorf("manual action definition %s is missing request url", m.ID)
	}
	if _, err := url.Parse(m.Request.URL); err != nil {
		return fmt.Errorf("manual action definition %s has invalid request url: %w", m.ID, err)
	}
	method := strings.ToUpper(strings.TrimSpace(m.Request.Method))
	if method == "" {
		return fmt.Errorf("manual action definition %s is missing request method", m.ID)
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
	default:
		return fmt.Errorf("manual action definition %s uses unsupported method %s", m.ID, m.Request.Method)
	}
	return nil
}

package model

import (
	"fmt"
)

// PipelineTabDefinition describes a UI tab for pipelines and the components it renders.
type PipelineTabDefinition struct {
	ID           string                     `json:"id"`
	Title        string                     `json:"title"`
	Icon         string                     `json:"icon,omitempty"`
	RequiresPush bool                       `json:"requires_push"`
	Components   []PipelineTabComponentMeta `json:"components"`
}

type PipelineTabComponentMeta struct {
	Type  string                 `json:"type"`
	Props map[string]interface{} `json:"props,omitempty"`
}

func (t PipelineTabDefinition) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("pipeline tab is missing id")
	}
	if t.Title == "" {
		return fmt.Errorf("pipeline tab %s is missing title", t.ID)
	}
	if len(t.Components) == 0 {
		return fmt.Errorf("pipeline tab %s must define at least one component", t.ID)
	}
	for _, cmp := range t.Components {
		if cmp.Type == "" {
			return fmt.Errorf("pipeline tab %s has component without type", t.ID)
		}
	}
	return nil
}

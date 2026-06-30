package model

import "time"

type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderAzure Provider = "azure"
	ProviderGCP   Provider = "gcp"
)

type DriftType string

const (
	DriftMissingInCloud   DriftType = "missing_in_cloud"
	DriftExtraInCloud     DriftType = "extra_in_cloud"
	DriftAttributeChanged DriftType = "attribute_changed"
	DriftTagsChanged      DriftType = "tags_changed"
)

// Resource is the canonical representation used for drift comparison.
type Resource struct {
	ID         string            `json:"id"`
	Provider   Provider          `json:"provider"`
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	Address    string            `json:"address,omitempty"`
	Region     string            `json:"region,omitempty"`
	Attributes map[string]any    `json:"attributes,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type Change struct {
	Path     string `json:"path"`
	Expected any    `json:"expected,omitempty"`
	Actual   any    `json:"actual,omitempty"`
}

type DriftItem struct {
	ResourceID   string    `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	DriftType    DriftType `json:"drift_type"`
	Changes      []Change  `json:"changes,omitempty"`
}

type Summary struct {
	TotalExpected int `json:"total_expected"`
	TotalActual   int `json:"total_actual"`
	Missing       int `json:"missing_in_cloud"`
	Extra         int `json:"extra_in_cloud"`
	Modified      int `json:"modified"`
	TagsChanged   int `json:"tags_changed"`
	Unchanged     int `json:"unchanged"`
}

type DriftReport struct {
	ScanID    string      `json:"scan_id"`
	Timestamp time.Time   `json:"timestamp"`
	Workspace string      `json:"workspace"`
	Summary   Summary     `json:"summary"`
	Drifts    []DriftItem `json:"drifts"`
}

package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/drift"
)

type IgnoreConfig struct {
	IgnoreAttributes []string `json:"ignore_attributes"`
	IgnoreTags       []string `json:"ignore_tags"`
	IgnorePaths      []string `json:"ignore_paths"`
}

func LoadIgnoreConfig(path string) (drift.Options, error) {
	if path == "" {
		return drift.Options{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return drift.Options{}, fmt.Errorf("read ignore config: %w", err)
	}

	var cfg IgnoreConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return drift.Options{}, fmt.Errorf("decode ignore config: %w", err)
	}

	return drift.Options{
		IgnoreAttributes: toSet(cfg.IgnoreAttributes),
		IgnoreTags:       toSet(cfg.IgnoreTags),
		IgnorePaths:      toSet(cfg.IgnorePaths),
	}, nil
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}

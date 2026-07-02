package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/config"
)

func TestLoadIgnoreConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignore.json")
	if err := os.WriteFile(path, []byte(`{
		"ignore_attributes": ["last_modified"],
		"ignore_tags": ["owner"],
		"ignore_paths": ["metadata.*"]
	}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	opts, err := config.LoadIgnoreConfig(path)
	if err != nil {
		t.Fatalf("LoadIgnoreConfig: %v", err)
	}

	if _, ok := opts.IgnoreAttributes["last_modified"]; !ok {
		t.Fatalf("expected last_modified ignore attribute")
	}
	if _, ok := opts.IgnoreTags["owner"]; !ok {
		t.Fatalf("expected owner ignore tag")
	}
	if _, ok := opts.IgnorePaths["metadata.*"]; !ok {
		t.Fatalf("expected metadata.* ignore path")
	}
}

package state_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/state"
)

func TestReadFile_ParsesManagedResources(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "terraform.tfstate")
	reader := state.NewReader()

	resources, err := reader.ReadFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}

	found := false
	for _, r := range resources {
		if r.Type == "aws_s3_bucket" && r.Name == "logs" {
			found = true
			if r.Attributes["bucket"] != "my-logs-bucket" {
				t.Fatalf("unexpected bucket attribute: %v", r.Attributes["bucket"])
			}
		}
	}
	if !found {
		t.Fatal("aws_s3_bucket.logs not found")
	}

	types := state.ResourceTypes(resources)
	if len(types) != 1 || types[0] != "aws_s3_bucket" {
		t.Fatalf("expected 1 aws_s3_bucket type, got %v", types)
	}
}

func TestReadFile_MissingFile(t *testing.T) {
	reader := state.NewReader()
	_, err := reader.ReadFile(context.Background(), filepath.Join(os.TempDir(), "missing.tfstate"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

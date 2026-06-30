package normalize_test

import (
	"testing"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/normalize"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/state"
)

func TestFromState_AWSS3Bucket(t *testing.T) {
	raw := []state.RawResource{{
		Address: "aws_s3_bucket.logs",
		Type:    "aws_s3_bucket",
		Name:    "logs",
		Attributes: map[string]any{
			"bucket": "my-logs-bucket",
			"region": "us-east-1",
			"tags": map[string]any{
				"Environment": "prod",
			},
			"versioning": []any{
				map[string]any{"enabled": true},
			},
		},
	}}

	resources := normalize.FromState(raw)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r := resources[0]
	if r.ID != "aws:aws_s3_bucket:us-east-1:logs" {
		t.Fatalf("unexpected id: %s", r.ID)
	}
	if r.Attributes["versioning"] != "Enabled" {
		t.Fatalf("expected Enabled versioning, got %v", r.Attributes["versioning"])
	}
	if r.Tags["Environment"] != "prod" {
		t.Fatalf("unexpected tags: %v", r.Tags)
	}
}

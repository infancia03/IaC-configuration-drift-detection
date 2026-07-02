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

func TestFromState_AWSAdditionalTypes(t *testing.T) {
	raw := []state.RawResource{
		{
			Address: "aws_instance.web",
			Type:    "aws_instance",
			Name:    "web",
			Attributes: map[string]any{
				"id":            "i-123",
				"ami":           "ami-123",
				"instance_type": "t3.micro",
				"tags": map[string]any{
					"Environment": "prod",
				},
			},
		},
		{
			Address: "aws_iam_role.app",
			Type:    "aws_iam_role",
			Name:    "app",
			Attributes: map[string]any{
				"name": "app-role",
				"arn":  "arn:aws:iam::123456789012:role/app-role",
			},
		},
		{
			Address: "aws_security_group.web",
			Type:    "aws_security_group",
			Name:    "web",
			Attributes: map[string]any{
				"id":          "sg-123",
				"name":        "web",
				"description": "web traffic",
				"ingress":     []any{map[string]any{"from_port": float64(443)}},
				"egress":      []any{map[string]any{"from_port": float64(0)}},
			},
		},
	}

	resources := normalize.FromState(raw)
	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	if resources[0].Attributes["cloud_id"] != "i-123" {
		t.Fatalf("unexpected instance cloud id: %v", resources[0].Attributes["cloud_id"])
	}
	if resources[1].Attributes["cloud_id"] != "app-role" {
		t.Fatalf("unexpected role cloud id: %v", resources[1].Attributes["cloud_id"])
	}
	if resources[2].Attributes["ingress_rule_count"] != 1 {
		t.Fatalf("unexpected ingress count: %v", resources[2].Attributes["ingress_rule_count"])
	}
}

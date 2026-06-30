package scan_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/scan"
)

func TestScanner_DryRunCloud(t *testing.T) {
	statePath := filepath.Join("..", "..", "testdata", "terraform.tfstate")
	scanner := scan.NewScanner()

	report, err := scanner.Run(context.Background(), scan.Options{
		Workspace:   "test",
		StatePath:   statePath,
		DryRunCloud: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Summary.TotalExpected != 2 {
		t.Fatalf("expected 2 resources in state, got %d", report.Summary.TotalExpected)
	}
	if len(report.Drifts) != 0 {
		t.Fatalf("dry-run should produce no drift, got %+v", report.Drifts)
	}
	if report.ScanID == "" {
		t.Fatal("expected scan_id in report")
	}
}

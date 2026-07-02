package drift_test

import (
	"testing"
	"time"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/drift"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
)

func TestCompare_MissingInCloud(t *testing.T) {
	model.Now = func() time.Time {
		return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		model.Now = func() time.Time { return time.Now().UTC() }
	})

	engine := drift.NewEngine(drift.Options{})
	expected := []model.Resource{
		{
			ID:       "aws:s3_bucket:us-east-1:logs",
			Provider: model.ProviderAWS,
			Type:     "aws_s3_bucket",
			Name:     "logs",
			Region:   "us-east-1",
			Attributes: map[string]any{
				"bucket": "my-logs-bucket",
			},
		},
	}

	report := engine.Compare("prod", expected, nil)

	if report.Summary.Missing != 1 {
		t.Fatalf("expected 1 missing, got %d", report.Summary.Missing)
	}
	if len(report.Drifts) != 1 || report.Drifts[0].DriftType != model.DriftMissingInCloud {
		t.Fatalf("unexpected drifts: %+v", report.Drifts)
	}
}

func TestCompare_ExtraInCloud(t *testing.T) {
	engine := drift.NewEngine(drift.Options{})
	actual := []model.Resource{
		{
			ID:       "aws:s3_bucket:us-east-1:orphan",
			Provider: model.ProviderAWS,
			Type:     "aws_s3_bucket",
			Name:     "orphan",
			Region:   "us-east-1",
			Attributes: map[string]any{
				"bucket": "orphan-bucket",
			},
		},
	}

	report := engine.Compare("prod", nil, actual)

	if report.Summary.Extra != 1 {
		t.Fatalf("expected 1 extra, got %d", report.Summary.Extra)
	}
	if report.Drifts[0].DriftType != model.DriftExtraInCloud {
		t.Fatalf("unexpected drift type: %s", report.Drifts[0].DriftType)
	}
}

func TestCompare_AttributeChanged(t *testing.T) {
	engine := drift.NewEngine(drift.Options{})
	id := "aws:s3_bucket:us-east-1:app"
	expected := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Attributes: map[string]any{"versioning": "Enabled"},
	}}
	actual := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Attributes: map[string]any{"versioning": "Suspended"},
	}}

	report := engine.Compare("prod", expected, actual)

	if report.Summary.Modified != 1 {
		t.Fatalf("expected 1 modified, got %d", report.Summary.Modified)
	}
	if report.Drifts[0].DriftType != model.DriftAttributeChanged {
		t.Fatalf("unexpected drift type: %s", report.Drifts[0].DriftType)
	}
	if len(report.Drifts[0].Changes) != 1 || report.Drifts[0].Changes[0].Path != "versioning" {
		t.Fatalf("unexpected changes: %+v", report.Drifts[0].Changes)
	}
}

func TestCompare_TagsChanged(t *testing.T) {
	engine := drift.NewEngine(drift.Options{})
	id := "aws:s3_bucket:us-east-1:app"
	expected := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Tags: map[string]string{"Environment": "prod"},
	}}
	actual := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Tags: map[string]string{"environment": "staging"},
	}}

	report := engine.Compare("prod", expected, actual)

	if report.Summary.TagsChanged != 1 {
		t.Fatalf("expected 1 tags_changed, got %d", report.Summary.TagsChanged)
	}
	if report.Drifts[0].DriftType != model.DriftTagsChanged {
		t.Fatalf("unexpected drift type: %s", report.Drifts[0].DriftType)
	}
}

func TestCompare_Unchanged(t *testing.T) {
	engine := drift.NewEngine(drift.Options{})
	id := "aws:s3_bucket:us-east-1:app"
	resource := model.Resource{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Attributes: map[string]any{"bucket": "my-bucket"},
		Tags:       map[string]string{"env": "prod"},
	}

	report := engine.Compare("prod", []model.Resource{resource}, []model.Resource{resource})

	if len(report.Drifts) != 0 {
		t.Fatalf("expected no drifts, got %+v", report.Drifts)
	}
	if report.Summary.Unchanged != 1 {
		t.Fatalf("expected 1 unchanged, got %d", report.Summary.Unchanged)
	}
}

func TestCompare_IgnoreAttributes(t *testing.T) {
	engine := drift.NewEngine(drift.Options{
		IgnoreAttributes: map[string]struct{}{
			"last_modified": {},
		},
	})
	id := "aws:s3_bucket:us-east-1:app"
	expected := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Attributes: map[string]any{"bucket": "same", "last_modified": "2026-01-01"},
	}}
	actual := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_s3_bucket", Name: "app", Region: "us-east-1",
		Attributes: map[string]any{"bucket": "same", "last_modified": "2026-06-01"},
	}}

	report := engine.Compare("prod", expected, actual)

	if len(report.Drifts) != 0 {
		t.Fatalf("expected ignored attribute to suppress drift, got %+v", report.Drifts)
	}
}

func TestCompare_IgnoreTagPaths(t *testing.T) {
	engine := drift.NewEngine(drift.Options{
		IgnoreTags: map[string]struct{}{
			"environment": {},
		},
	})
	id := "aws:aws_instance:us-east-1:web"
	expected := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_instance", Name: "web", Region: "us-east-1",
		Tags: map[string]string{"Environment": "prod"},
	}}
	actual := []model.Resource{{
		ID: id, Provider: model.ProviderAWS, Type: "aws_instance", Name: "web", Region: "us-east-1",
		Tags: map[string]string{"environment": "staging"},
	}}

	report := engine.Compare("prod", expected, actual)

	if len(report.Drifts) != 0 {
		t.Fatalf("expected ignored tag to suppress drift, got %+v", report.Drifts)
	}
}

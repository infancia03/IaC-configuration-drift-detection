package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/storage"
)

func TestStore_SaveListLatestAndGet(t *testing.T) {
	store := storage.NewStore(filepath.Join(t.TempDir(), "history.json"))
	older := model.DriftReport{ScanID: "older", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	newer := model.DriftReport{ScanID: "newer", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}

	if err := store.Save(older); err != nil {
		t.Fatalf("Save older: %v", err)
	}
	if err := store.Save(newer); err != nil {
		t.Fatalf("Save newer: %v", err)
	}

	reports, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(reports) != 2 || reports[0].ScanID != "newer" {
		t.Fatalf("expected newest report first, got %+v", reports)
	}

	latest, ok, err := store.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if !ok || latest.ScanID != "newer" {
		t.Fatalf("unexpected latest: ok=%v report=%+v", ok, latest)
	}

	got, ok, err := store.Get("older")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || got.ScanID != "older" {
		t.Fatalf("unexpected get result: ok=%v report=%+v", ok, got)
	}
}

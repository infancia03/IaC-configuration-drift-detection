package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) *Store {
	if path == "" {
		path = "drift-history.json"
	}
	return &Store{path: path}
}

func (s *Store) Save(report model.DriftReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reports, err := s.load()
	if err != nil {
		return err
	}
	reports = append(reports, report)
	sortReports(reports)
	return s.write(reports)
}

func (s *Store) List() ([]model.DriftReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) Latest() (model.DriftReport, bool, error) {
	reports, err := s.List()
	if err != nil {
		return model.DriftReport{}, false, err
	}
	if len(reports) == 0 {
		return model.DriftReport{}, false, nil
	}
	return reports[0], true, nil
}

func (s *Store) Get(scanID string) (model.DriftReport, bool, error) {
	reports, err := s.List()
	if err != nil {
		return model.DriftReport{}, false, err
	}
	for _, report := range reports {
		if report.ScanID == scanID {
			return report, true, nil
		}
	}
	return model.DriftReport{}, false, nil
}

func (s *Store) load() ([]model.DriftReport, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.DriftReport{}, nil
		}
		return nil, fmt.Errorf("read history: %w", err)
	}
	if len(data) == 0 {
		return []model.DriftReport{}, nil
	}

	var reports []model.DriftReport
	if err := json.Unmarshal(data, &reports); err != nil {
		return nil, fmt.Errorf("decode history: %w", err)
	}
	sortReports(reports)
	return reports, nil
}

func (s *Store) write(reports []model.DriftReport) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil && filepath.Dir(s.path) != "." {
		return fmt.Errorf("create history dir: %w", err)
	}
	data, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return fmt.Errorf("encode history: %w", err)
	}
	return os.WriteFile(s.path, data, 0o600)
}

func sortReports(reports []model.DriftReport) {
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Timestamp.After(reports[j].Timestamp)
	})
}

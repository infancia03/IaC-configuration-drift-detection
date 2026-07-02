package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/drift"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/scan"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/scheduler"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/storage"
)

type Server struct {
	scanner       *scan.Scanner
	store         *storage.Store
	defaultScan   scan.Options
	scheduleEvery time.Duration
}

type Options struct {
	Store         *storage.Store
	DefaultScan   scan.Options
	ScheduleEvery time.Duration
}

type ScanRequest struct {
	Workspace        string   `json:"workspace"`
	StatePath        string   `json:"state_path"`
	StateS3Bucket    string   `json:"state_s3_bucket"`
	StateS3Key       string   `json:"state_s3_key"`
	StateS3Region    string   `json:"state_s3_region"`
	AWSRegion        string   `json:"aws_region"`
	AWSProfile       string   `json:"aws_profile"`
	DryRunCloud      bool     `json:"dry_run_cloud"`
	IgnoreAttributes []string `json:"ignore_attributes"`
	IgnoreTags       []string `json:"ignore_tags"`
	IgnorePaths      []string `json:"ignore_paths"`
}

func NewServer(opts Options) *Server {
	return &Server{
		scanner:       scan.NewScanner(),
		store:         opts.Store,
		defaultScan:   opts.DefaultScan,
		scheduleEvery: opts.ScheduleEvery,
	}
}

func (s *Server) Handler(ctx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.dashboard)
	mux.HandleFunc("/api/health", s.health)
	mux.HandleFunc("/api/scans", s.scans)
	mux.HandleFunc("/api/scans/", s.scanByID)
	mux.HandleFunc("/api/reports/latest", s.latest)

	scheduler.Runner{
		Interval: s.scheduleEvery,
		Job: func(jobCtx context.Context) error {
			report, err := s.scanner.Run(jobCtx, s.defaultScan)
			if err != nil {
				return err
			}
			return s.store.Save(report)
		},
	}.Start(ctx)

	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) scans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		reports, err := s.store.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, reports)
	case http.MethodPost:
		var req ScanRequest
		if r.Body != nil {
			defer r.Body.Close()
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, fmt.Errorf("decode scan request: %w", err))
			return
		}
		opts := mergeScanOptions(s.defaultScan, req)
		report, err := s.scanner.Run(r.Context(), opts)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		if err := s.store.Save(report); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, report)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) scanByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/scans/")
	report, ok, err := s.store.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("scan %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) latest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	report, ok, err := s.store.Latest()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("no scans found"))
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = dashboardTemplate.Execute(w, nil)
}

func mergeScanOptions(base scan.Options, req ScanRequest) scan.Options {
	opts := base
	if req.Workspace != "" {
		opts.Workspace = req.Workspace
	}
	if req.StatePath != "" {
		opts.StatePath = req.StatePath
	}
	if req.StateS3Bucket != "" {
		opts.StateS3Bucket = req.StateS3Bucket
	}
	if req.StateS3Key != "" {
		opts.StateS3Key = req.StateS3Key
	}
	if req.StateS3Region != "" {
		opts.StateS3Region = req.StateS3Region
	}
	if req.AWSRegion != "" {
		opts.AWSRegion = req.AWSRegion
	}
	if req.AWSProfile != "" {
		opts.AWSProfile = req.AWSProfile
	}
	if req.DryRunCloud {
		opts.DryRunCloud = true
	}
	if len(req.IgnoreAttributes) > 0 || len(req.IgnoreTags) > 0 || len(req.IgnorePaths) > 0 {
		opts.Drift = drift.Options{
			IgnoreAttributes: toSet(req.IgnoreAttributes),
			IgnoreTags:       toSet(req.IgnoreTags),
			IgnorePaths:      toSet(req.IgnorePaths),
		}
	}
	return opts
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Drift Detection</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { margin: 0; background: #f7f8fb; color: #172033; }
    header { padding: 24px 32px; background: #ffffff; border-bottom: 1px solid #dfe4ec; display: flex; align-items: center; justify-content: space-between; gap: 16px; }
    h1 { margin: 0; font-size: 22px; font-weight: 700; }
    main { max-width: 1180px; margin: 0 auto; padding: 24px; }
    .toolbar { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
    button { border: 1px solid #1c5f99; background: #1c5f99; color: #fff; border-radius: 6px; padding: 9px 13px; font-weight: 650; cursor: pointer; }
    button.secondary { background: #fff; color: #1c5f99; }
    .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin: 20px 0; }
    .tile, .panel { background: #fff; border: 1px solid #dfe4ec; border-radius: 8px; }
    .tile { padding: 16px; }
    .tile span { display: block; color: #64748b; font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    .tile strong { display: block; margin-top: 8px; font-size: 24px; }
    .panel { overflow: hidden; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 12px 14px; border-bottom: 1px solid #edf0f4; text-align: left; font-size: 14px; vertical-align: top; }
    th { background: #fbfcfe; color: #475569; font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    code { background: #eef2f6; border-radius: 4px; padding: 2px 5px; }
    .empty { padding: 28px; color: #64748b; }
  </style>
</head>
<body>
  <header>
    <h1>Terraform Drift Detection</h1>
    <div class="toolbar">
      <button onclick="runScan()">Run Scan</button>
      <button class="secondary" onclick="loadLatest()">Refresh</button>
    </div>
  </header>
  <main>
    <section class="summary" id="summary"></section>
    <section class="panel">
      <table>
        <thead><tr><th>Resource</th><th>Type</th><th>Drift</th><th>Changes</th></tr></thead>
        <tbody id="drifts"><tr><td class="empty" colspan="4">No scan loaded.</td></tr></tbody>
      </table>
    </section>
  </main>
  <script>
    async function loadLatest() {
      const res = await fetch('/api/reports/latest');
      if (!res.ok) {
        document.getElementById('summary').innerHTML = '<div class="tile"><span>Status</span><strong>No scans</strong></div>';
        return;
      }
      render(await res.json());
    }
    async function runScan() {
      const res = await fetch('/api/scans', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: '{}'});
      const body = await res.json();
      if (!res.ok) { alert(body.error || 'Scan failed'); return; }
      render(body);
    }
    function render(report) {
      const s = report.summary || {};
      const tiles = [['Expected', s.total_expected], ['Actual', s.total_actual], ['Missing', s.missing_in_cloud], ['Extra', s.extra_in_cloud], ['Modified', s.modified], ['Tags', s.tags_changed], ['Unchanged', s.unchanged]];
      document.getElementById('summary').innerHTML = tiles.map(([k,v]) => '<div class="tile"><span>'+k+'</span><strong>'+(v || 0)+'</strong></div>').join('');
      const drifts = report.drifts || [];
      document.getElementById('drifts').innerHTML = drifts.length ? drifts.map(d => '<tr><td><code>'+escapeHTML(d.resource_name || d.resource_id)+'</code></td><td>'+escapeHTML(d.resource_type)+'</td><td>'+escapeHTML(d.drift_type)+'</td><td>'+escapeHTML((d.changes || []).map(c => c.path).join(', '))+'</td></tr>').join('') : '<tr><td class="empty" colspan="4">No drift detected.</td></tr>';
    }
    function escapeHTML(value) {
      return String(value || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
    }
    loadLatest();
  </script>
</body>
</html>`))

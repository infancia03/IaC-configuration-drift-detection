# IaC Configuration Drift Detection

Cloud-agnostic Terraform drift detection platform. Compares Terraform state (expected) against live cloud infrastructure (actual), normalizes both into a common model, and reports differences without running `terraform plan` or `apply`.

## Phase 1 vertical slice

This MVP implements the first end-to-end path for AWS S3 buckets:

```
Terraform State → State Reader → Normalizer → Expected Model
Cloud APIs      → AWS Fetcher → Normalizer → Actual Model
                                              ↓
                                        Drift Engine
                                              ↓
                                      JSON Report (CLI)
```

## Quick start

### Prerequisites

- Go 1.22+
- AWS credentials configured (for live scans)
- A Terraform state file with `aws_s3_bucket` resources

### Run tests

```bash
go test ./...
```

### Scan with JSON output

```bash
# Live AWS comparison
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --output json

# Local dry-run (no AWS calls; validates pipeline)
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --dry-run-cloud --output json
```

### Build binary

```bash
go build -o bin/driftctl ./cmd/driftctl
```

## Project layout

```
cmd/driftctl/          CLI entrypoint
internal/model/        Canonical Resource and DriftReport types
internal/drift/        Compare engine
internal/state/        Terraform state reader
internal/normalize/    State/cloud → canonical model mappers
internal/providers/aws AWS resource fetcher
internal/scan/         Orchestration pipeline
internal/report/       JSON report writer
testdata/              Fixture state for tests
```

## Drift types

| Type | Meaning |
|------|---------|
| `missing_in_cloud` | Resource in state but not found in cloud |
| `extra_in_cloud` | Resource in cloud but not in state |
| `attribute_changed` | Comparable attribute differs |
| `tags_changed` | Tag key/value differs |

## Roadmap

- [ ] S3 remote state backend
- [ ] Additional AWS resource types (EC2, IAM, RDS)
- [ ] Azure and GCP providers
- [ ] Scheduled scans and REST API
- [ ] Dashboard UI

## License

MIT

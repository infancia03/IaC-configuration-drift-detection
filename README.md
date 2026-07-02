# IaC Configuration Drift Detection

Cloud-agnostic Terraform drift detection platform. It compares Terraform state (expected) against live cloud infrastructure (actual), normalizes both into a common model, and reports differences without running `terraform plan` or `terraform apply`.

## Complete Stack

This implementation provides a local-first AWS drift detection stack with CLI output, JSON reports, file-backed scan history, scheduled scans, a REST API, and a simple dashboard.

```
Terraform State -> State Reader -> Normalizer -> Expected Model
Cloud APIs      -> AWS Fetcher  -> Normalizer -> Actual Model
                                               |
                                         Drift Engine
                                               |
                         CLI / JSON / History / API / Dashboard
```

## Supported Resources

- `aws_s3_bucket`
- `aws_instance`
- `aws_security_group`
- `aws_iam_role`
- `aws_db_instance`
- `aws_lambda_function`
- `aws_eks_cluster`

## Quick Start

### Prerequisites

- Go 1.22+
- AWS credentials configured for live scans
- A Terraform state file with supported AWS resources

### Run Tests

```bash
go mod tidy
go test ./...
```

### CLI Scans

```bash
# Live AWS comparison
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --output json

# Remote S3 state backend
go run ./cmd/driftctl scan \
  --state-s3-bucket my-tfstate-bucket \
  --state-s3-key envs/prod/terraform.tfstate \
  --state-s3-region us-east-1 \
  --output json

# Local dry-run with no AWS calls
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --dry-run-cloud --output json

# Persist scan history
go run ./cmd/driftctl scan \
  --state ./testdata/terraform.tfstate \
  --history-file ./drift-history.json \
  --output json

# CI mode: fail with a non-zero exit code when drift is detected
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --fail-on-drift --output json
```

### Ignore Rules

Create a JSON ignore rules file for noisy attributes or tags:

```json
{
  "ignore_attributes": ["last_modified", "public_ip"],
  "ignore_tags": ["owner"],
  "ignore_paths": ["metadata.*"]
}
```

Then pass it to scans:

```bash
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --ignore-config ./ignore.json --output json
```

### Scan History

```bash
go run ./cmd/driftctl history --history-file ./drift-history.json --limit 10
```

### API, Dashboard, And Scheduled Scans

```bash
go run ./cmd/driftctl serve \
  --state ./testdata/terraform.tfstate \
  --history-file ./drift-history.json \
  --addr 127.0.0.1:8080 \
  --schedule-every 15m
```

Open `http://127.0.0.1:8080` for the dashboard.

REST endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/health` | Health check |
| `POST` | `/api/scans` | Run a scan and persist the report |
| `GET` | `/api/scans` | List scan history |
| `GET` | `/api/scans/{scan_id}` | Fetch one report |
| `GET` | `/api/reports/latest` | Fetch latest report |

### Build Binary

```bash
go build -o bin/driftctl ./cmd/driftctl
```

## Project Layout

```text
cmd/driftctl/          CLI entrypoint
internal/api/          REST API and embedded dashboard
internal/config/       Ignore rules parser
internal/drift/        Compare engine
internal/model/        Canonical Resource and DriftReport types
internal/normalize/    State/cloud to canonical model mappers
internal/providers/aws AWS resource fetcher
internal/report/       JSON report writer
internal/scan/         Orchestration pipeline
internal/scheduler/    Interval scan runner
internal/state/        Terraform state reader
internal/storage/      File-backed scan history
testdata/              Fixture state for tests
```

## Drift Types

| Type | Meaning |
|------|---------|
| `missing_in_cloud` | Resource in state but not found in cloud |
| `extra_in_cloud` | Resource in cloud but not in state |
| `attribute_changed` | Comparable attribute differs |
| `tags_changed` | Tag key/value differs |

## Roadmap

- [x] S3 remote state backend
- [x] AWS S3, EC2, security group, IAM role, RDS, Lambda, and EKS support
- [x] CI failure mode with `--fail-on-drift`
- [x] Ignore rules config file
- [x] Scheduled scans and REST API
- [x] Dashboard UI
- [x] Scan history storage
- [x] GitHub Actions CI
- [ ] Azure and GCP providers

## License

MIT

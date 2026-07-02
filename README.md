# IaC Configuration Drift Detection

Cloud-agnostic Terraform drift detection platform. Compares Terraform state (expected) against live cloud infrastructure (actual), normalizes both into a common model, and reports differences without running `terraform plan` or `apply`.

## Phase 2 vertical slice

This MVP implements the first end-to-end path for AWS S3 buckets, EC2 instances, IAM roles, and security groups:

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
- A Terraform state file with supported AWS resources

### Run tests

```bash
go test ./...
```

### Scan with JSON output

```bash
# Live AWS comparison
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --output json

# Remote S3 state backend
go run ./cmd/driftctl scan \
  --state-s3-bucket my-tfstate-bucket \
  --state-s3-key envs/prod/terraform.tfstate \
  --state-s3-region us-east-1 \
  --output json

# Local dry-run (no AWS calls; validates pipeline)
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --dry-run-cloud --output json

# CI mode: fail with a non-zero exit code when drift is detected
go run ./cmd/driftctl scan --state ./testdata/terraform.tfstate --fail-on-drift --output json
```

### Ignore rules

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

- [x] S3 remote state backend
- [x] Additional AWS resource types (EC2, IAM, security groups)
- [x] CI failure mode with `--fail-on-drift`
- [x] Ignore rules config file
- [ ] RDS support
- [ ] Azure and GCP providers
- [ ] Scheduled scans and REST API
- [ ] Dashboard UI

## License

MIT

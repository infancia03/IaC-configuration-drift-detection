package scan

import (
	"context"
	"fmt"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/drift"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/normalize"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/providers/aws"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/state"
)

type Options struct {
	Workspace     string
	StatePath     string
	StateS3Bucket string
	StateS3Key    string
	StateS3Region string
	AWSRegion     string
	AWSProfile    string
	Drift         drift.Options
	DryRunCloud   bool
}

type Scanner struct {
	stateReader *state.Reader
}

func NewScanner() *Scanner {
	return &Scanner{stateReader: state.NewReader()}
}

// Run executes the full state → fetch → compare pipeline.
func (s *Scanner) Run(ctx context.Context, opts Options) (model.DriftReport, error) {
	if opts.StatePath == "" && (opts.StateS3Bucket == "" || opts.StateS3Key == "") {
		return model.DriftReport{}, fmt.Errorf("state path or s3 state bucket/key is required")
	}
	if opts.Workspace == "" {
		opts.Workspace = "default"
	}

	rawState, err := s.readState(ctx, opts)
	if err != nil {
		return model.DriftReport{}, fmt.Errorf("read state: %w", err)
	}

	expected := normalize.FromState(rawState)
	if len(expected) == 0 {
		return model.DriftReport{}, fmt.Errorf("no supported managed resources found in state")
	}

	var actual []model.Resource
	if opts.DryRunCloud {
		actual = expected
	} else {
		fetcher := aws.NewFetcher(aws.Options{
			Region:  opts.AWSRegion,
			Profile: opts.AWSProfile,
		})
		cloudRaw, err := fetcher.Fetch(ctx, state.ResourceTypes(rawState))
		if err != nil {
			return model.DriftReport{}, fmt.Errorf("fetch cloud resources: %w", err)
		}
		actual = alignCloudResources(expected, normalize.FromCloud(cloudRaw))
	}

	engine := drift.NewEngine(opts.Drift)
	report := engine.Compare(opts.Workspace, expected, actual)
	return report, nil
}

func (s *Scanner) readState(ctx context.Context, opts Options) ([]state.RawResource, error) {
	if opts.StateS3Bucket != "" || opts.StateS3Key != "" {
		region := opts.StateS3Region
		if region == "" {
			region = opts.AWSRegion
		}
		return s.stateReader.ReadS3(ctx, state.S3Backend{
			Bucket:  opts.StateS3Bucket,
			Key:     opts.StateS3Key,
			Region:  region,
			Profile: opts.AWSProfile,
		})
	}
	return s.stateReader.ReadFile(ctx, opts.StatePath)
}

// alignCloudResources maps cloud resources to expected canonical IDs using cloud identifiers.
func alignCloudResources(expected, actual []model.Resource) []model.Resource {
	cloudIDToExpected := make(map[string]model.Resource)
	for _, exp := range expected {
		for _, key := range []string{"cloud_id", "bucket"} {
			if id, ok := exp.Attributes[key].(string); ok && id != "" {
				cloudIDToExpected[identityKey(exp.Type, key, id)] = exp
			}
		}
	}

	aligned := make([]model.Resource, 0, len(actual))
	for _, act := range actual {
		matched := false
		for _, key := range []string{"cloud_id", "bucket"} {
			id, _ := act.Attributes[key].(string)
			if id == "" {
				continue
			}
			if exp, ok := cloudIDToExpected[identityKey(act.Type, key, id)]; ok {
				act.ID = exp.ID
				act.Name = exp.Name
				act.Address = exp.Address
				act.Region = exp.Region
				matched = true
				break
			}
		}
		if !matched {
			id, _ := act.Attributes["cloud_id"].(string)
			if id == "" {
				id, _ = act.Attributes["bucket"].(string)
			}
			act.ID = fmt.Sprintf("%s:%s:%s:cloud:%s", act.Provider, act.Type, act.Region, id)
		}
		aligned = append(aligned, act)
	}
	return aligned
}

func identityKey(resourceType, attr, id string) string {
	return resourceType + ":" + attr + ":" + id
}

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
	Workspace string
	StatePath string
	AWSRegion string
	AWSProfile string
	Drift      drift.Options
	DryRunCloud bool
}

type Scanner struct {
	stateReader *state.Reader
}

func NewScanner() *Scanner {
	return &Scanner{stateReader: state.NewReader()}
}

// Run executes the full state → fetch → compare pipeline.
func (s *Scanner) Run(ctx context.Context, opts Options) (model.DriftReport, error) {
	if opts.StatePath == "" {
		return model.DriftReport{}, fmt.Errorf("state path is required")
	}
	if opts.Workspace == "" {
		opts.Workspace = "default"
	}

	rawState, err := s.stateReader.ReadFile(ctx, opts.StatePath)
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

// alignCloudResources maps cloud resources to expected canonical IDs using bucket name.
func alignCloudResources(expected, actual []model.Resource) []model.Resource {
	bucketToExpected := make(map[string]model.Resource)
	for _, exp := range expected {
		if bucket, ok := exp.Attributes["bucket"].(string); ok {
			bucketToExpected[bucket] = exp
		}
	}

	aligned := make([]model.Resource, 0, len(actual))
	for _, act := range actual {
		bucket, _ := act.Attributes["bucket"].(string)
		if exp, ok := bucketToExpected[bucket]; ok {
			act.ID = exp.ID
			act.Name = exp.Name
			act.Address = exp.Address
			act.Region = exp.Region
		} else {
			act.ID = fmt.Sprintf("%s:%s:%s:bucket:%s", act.Provider, act.Type, act.Region, bucket)
		}
		aligned = append(aligned, act)
	}
	return aligned
}

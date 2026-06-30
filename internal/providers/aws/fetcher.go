package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Resource is a cloud-fetched resource before normalization.
type Resource struct {
	Type        string
	Name        string
	BucketName  string
	Region      string
	Versioning  string
	Tags        map[string]string
}

// Fetcher retrieves live AWS resources referenced by Terraform state.
type Fetcher struct {
	region  string
	profile string
}

type Options struct {
	Region  string
	Profile string
}

func NewFetcher(opts Options) *Fetcher {
	region := opts.Region
	if region == "" {
		region = "us-east-1"
	}
	return &Fetcher{region: region, profile: opts.Profile}
}

// Fetch loads AWS resources for the requested Terraform resource types.
func (f *Fetcher) Fetch(ctx context.Context, resourceTypes []string) ([]Resource, error) {
	wantS3 := false
	for _, t := range resourceTypes {
		if t == "aws_s3_bucket" {
			wantS3 = true
			break
		}
	}
	if !wantS3 {
		return nil, nil
	}
	return f.fetchS3Buckets(ctx)
}

func (f *Fetcher) loadConfig(ctx context.Context) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(f.region),
	}
	if f.profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(f.profile))
	}
	return config.LoadDefaultConfig(ctx, opts...)
}

func (f *Fetcher) fetchS3Buckets(ctx context.Context) ([]Resource, error) {
	cfg, err := f.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	listOut, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("list s3 buckets: %w", err)
	}

	out := make([]Resource, 0, len(listOut.Buckets))
	for _, bucket := range listOut.Buckets {
		if bucket.Name == nil {
			continue
		}
		name := aws.ToString(bucket.Name)

		verOut, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: bucket.Name,
		})
		versioning := "Disabled"
		if err == nil && verOut.Status == "Enabled" {
			versioning = "Enabled"
		}

		tags, _ := f.fetchBucketTags(ctx, client, bucket.Name)

		out = append(out, Resource{
			Type:       "aws_s3_bucket",
			BucketName: name,
			Region:     f.region,
			Versioning: versioning,
			Tags:       tags,
		})
	}
	return out, nil
}

func (f *Fetcher) fetchBucketTags(ctx context.Context, client *s3.Client, bucket *string) (map[string]string, error) {
	out, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: bucket})
	if err != nil {
		return map[string]string{}, nil
	}
	tags := make(map[string]string, len(out.TagSet))
	for _, tag := range out.TagSet {
		if tag.Key != nil && tag.Value != nil {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}
	return tags, nil
}

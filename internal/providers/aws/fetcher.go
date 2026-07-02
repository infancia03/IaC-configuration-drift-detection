package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Resource is a cloud-fetched resource before normalization.
type Resource struct {
	Type               string
	Name               string
	BucketName         string
	Region             string
	Versioning         string
	CloudID            string
	ARN                string
	State              string
	InstanceType       string
	ImageID            string
	PrivateIPAddress   string
	PublicIPAddress    string
	SubnetID           string
	VPCID              string
	SecurityGroupName  string
	Description        string
	IngressRuleCount   int
	EgressRuleCount    int
	Path               string
	AssumeRolePolicy   string
	Tags               map[string]string
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
	want := wantedTypes(resourceTypes)
	if len(want) == 0 {
		return nil, nil
	}

	cfg, err := f.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	out := make([]Resource, 0)
	if want["aws_s3_bucket"] {
		resources, err := f.fetchS3Buckets(ctx, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, resources...)
	}
	if want["aws_instance"] {
		resources, err := f.fetchEC2Instances(ctx, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, resources...)
	}
	if want["aws_security_group"] {
		resources, err := f.fetchSecurityGroups(ctx, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, resources...)
	}
	if want["aws_iam_role"] {
		resources, err := f.fetchIAMRoles(ctx, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, resources...)
	}
	return out, nil
}

func wantedTypes(resourceTypes []string) map[string]bool {
	out := map[string]bool{}
	for _, t := range resourceTypes {
		switch t {
		case "aws_s3_bucket", "aws_instance", "aws_iam_role", "aws_security_group":
			out[t] = true
		}
	}
	return out
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

func (f *Fetcher) fetchS3Buckets(ctx context.Context, cfg aws.Config) ([]Resource, error) {
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

func (f *Fetcher) fetchEC2Instances(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	out := make([]Resource, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe ec2 instances: %w", err)
		}
		for _, reservation := range page.Reservations {
			for _, inst := range reservation.Instances {
				out = append(out, Resource{
					Type:             "aws_instance",
					Name:             tagValue(inst.Tags, "Name"),
					Region:           f.region,
					CloudID:          aws.ToString(inst.InstanceId),
					State:            string(inst.State.Name),
					InstanceType:     string(inst.InstanceType),
					ImageID:          aws.ToString(inst.ImageId),
					PrivateIPAddress: aws.ToString(inst.PrivateIpAddress),
					PublicIPAddress:  aws.ToString(inst.PublicIpAddress),
					SubnetID:         aws.ToString(inst.SubnetId),
					VPCID:            aws.ToString(inst.VpcId),
					Tags:             ec2Tags(inst.Tags),
				})
			}
		}
	}
	return out, nil
}

func (f *Fetcher) fetchSecurityGroups(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{})

	out := make([]Resource, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe security groups: %w", err)
		}
		for _, sg := range page.SecurityGroups {
			out = append(out, Resource{
				Type:              "aws_security_group",
				Name:              aws.ToString(sg.GroupName),
				Region:            f.region,
				CloudID:           aws.ToString(sg.GroupId),
				SecurityGroupName: aws.ToString(sg.GroupName),
				Description:       aws.ToString(sg.Description),
				VPCID:             aws.ToString(sg.VpcId),
				IngressRuleCount:  len(sg.IpPermissions),
				EgressRuleCount:   len(sg.IpPermissionsEgress),
				Tags:              ec2Tags(sg.Tags),
			})
		}
	}
	return out, nil
}

func (f *Fetcher) fetchIAMRoles(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})

	out := make([]Resource, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list iam roles: %w", err)
		}
		for _, role := range page.Roles {
			tags, _ := f.fetchRoleTags(ctx, client, aws.ToString(role.RoleName))
			out = append(out, Resource{
				Type:             "aws_iam_role",
				Name:             aws.ToString(role.RoleName),
				Region:           f.region,
				CloudID:          aws.ToString(role.RoleName),
				ARN:              aws.ToString(role.Arn),
				Path:             aws.ToString(role.Path),
				AssumeRolePolicy: aws.ToString(role.AssumeRolePolicyDocument),
				Tags:             tags,
			})
		}
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

func (f *Fetcher) fetchRoleTags(ctx context.Context, client *iam.Client, roleName string) (map[string]string, error) {
	out, err := client.ListRoleTags(ctx, &iam.ListRoleTagsInput{RoleName: aws.String(roleName)})
	if err != nil {
		return map[string]string{}, nil
	}
	tags := make(map[string]string, len(out.Tags))
	for _, tag := range out.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}
	return tags, nil
}

func ec2Tags(tags []ec2types.Tag) map[string]string {
	out := make(map[string]string, len(tags))
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			out[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}
	return out
}

func tagValue(tags []ec2types.Tag, key string) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == key {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

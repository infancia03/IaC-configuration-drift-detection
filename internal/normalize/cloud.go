package normalize

import (
	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/providers/aws"
)

// FromCloud converts provider-fetched resources into canonical resources.
func FromCloud(raw []aws.Resource) []model.Resource {
	out := make([]model.Resource, 0, len(raw))
	for _, r := range raw {
		switch r.Type {
		case "aws_s3_bucket":
			out = append(out, mapAWSS3FromCloud(r))
		case "aws_instance":
			out = append(out, mapAWSInstanceFromCloud(r))
		case "aws_iam_role":
			out = append(out, mapAWSIAMRoleFromCloud(r))
		case "aws_security_group":
			out = append(out, mapAWSSecurityGroupFromCloud(r))
		}
	}
	return out
}

func mapAWSInstanceFromCloud(r aws.Resource) model.Resource {
	name := r.Name
	if name == "" {
		name = r.CloudID
	}
	return model.Resource{
		ID:       canonicalCloudID("aws_instance", r.Region, r.CloudID),
		Provider: model.ProviderAWS,
		Type:     "aws_instance",
		Name:     name,
		Region:   r.Region,
		Attributes: map[string]any{
			"cloud_id":      r.CloudID,
			"ami":           r.ImageID,
			"instance_type": r.InstanceType,
			"state":         r.State,
			"subnet_id":     r.SubnetID,
			"vpc_id":        r.VPCID,
			"private_ip":    r.PrivateIPAddress,
			"public_ip":     r.PublicIPAddress,
		},
		Tags: r.Tags,
	}
}

func mapAWSIAMRoleFromCloud(r aws.Resource) model.Resource {
	return model.Resource{
		ID:       canonicalCloudID("aws_iam_role", r.Region, r.CloudID),
		Provider: model.ProviderAWS,
		Type:     "aws_iam_role",
		Name:     r.Name,
		Region:   r.Region,
		Attributes: map[string]any{
			"cloud_id":           r.CloudID,
			"name":               r.Name,
			"arn":                r.ARN,
			"path":               r.Path,
			"assume_role_policy": r.AssumeRolePolicy,
		},
		Tags: r.Tags,
	}
}

func mapAWSSecurityGroupFromCloud(r aws.Resource) model.Resource {
	return model.Resource{
		ID:       canonicalCloudID("aws_security_group", r.Region, r.CloudID),
		Provider: model.ProviderAWS,
		Type:     "aws_security_group",
		Name:     r.SecurityGroupName,
		Region:   r.Region,
		Attributes: map[string]any{
			"cloud_id":           r.CloudID,
			"name":               r.SecurityGroupName,
			"description":        r.Description,
			"vpc_id":             r.VPCID,
			"ingress_rule_count": r.IngressRuleCount,
			"egress_rule_count":  r.EgressRuleCount,
		},
		Tags: r.Tags,
	}
}

func canonicalCloudID(resourceType, region, cloudID string) string {
	if region == "" {
		region = "us-east-1"
	}
	return canonicalID(model.ProviderAWS, resourceType, region, cloudID)
}

func mapAWSS3FromCloud(r aws.Resource) model.Resource {
	region := r.Region
	if region == "" {
		region = "us-east-1"
	}

	tfName := r.Name
	if tfName == "" {
		tfName = MatchNameFromBucket(r.Tags, r.BucketName)
	}

	return model.Resource{
		ID:       CanonicalBucketID(region, tfName),
		Provider: model.ProviderAWS,
		Type:     "aws_s3_bucket",
		Name:     tfName,
		Region:   region,
		Attributes: map[string]any{
			"bucket":     r.BucketName,
			"versioning": r.Versioning,
		},
		Tags: r.Tags,
	}
}

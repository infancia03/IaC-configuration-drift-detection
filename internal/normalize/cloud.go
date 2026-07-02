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
		case "aws_db_instance":
			out = append(out, mapAWSDBInstanceFromCloud(r))
		case "aws_lambda_function":
			out = append(out, mapAWSLambdaFunctionFromCloud(r))
		case "aws_eks_cluster":
			out = append(out, mapAWSEKSClusterFromCloud(r))
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
			"subnet_id":     r.SubnetID,
			"vpc_id":        r.VPCID,
			"private_ip":    r.PrivateIPAddress,
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

func mapAWSDBInstanceFromCloud(r aws.Resource) model.Resource {
	return model.Resource{
		ID:       canonicalCloudID("aws_db_instance", r.Region, r.CloudID),
		Provider: model.ProviderAWS,
		Type:     "aws_db_instance",
		Name:     r.Name,
		Region:   r.Region,
		Attributes: map[string]any{
			"cloud_id":            r.CloudID,
			"identifier":          r.CloudID,
			"arn":                 r.ARN,
			"engine":              r.Engine,
			"instance_class":      r.DBInstanceClass,
			"allocated_storage":   int(r.AllocatedStorage),
			"storage_type":        r.StorageType,
			"multi_az":            r.MultiAZ,
			"publicly_accessible": r.PubliclyAccessible,
		},
		Tags: r.Tags,
	}
}

func mapAWSLambdaFunctionFromCloud(r aws.Resource) model.Resource {
	return model.Resource{
		ID:       canonicalCloudID("aws_lambda_function", r.Region, r.CloudID),
		Provider: model.ProviderAWS,
		Type:     "aws_lambda_function",
		Name:     r.Name,
		Region:   r.Region,
		Attributes: map[string]any{
			"cloud_id":      r.CloudID,
			"function_name": r.Name,
			"arn":           r.ARN,
			"runtime":       r.Runtime,
			"handler":       r.Handler,
			"role":          r.Role,
			"memory_size":   int(r.MemorySize),
			"timeout":       int(r.Timeout),
		},
		Tags: r.Tags,
	}
}

func mapAWSEKSClusterFromCloud(r aws.Resource) model.Resource {
	return model.Resource{
		ID:       canonicalCloudID("aws_eks_cluster", r.Region, r.CloudID),
		Provider: model.ProviderAWS,
		Type:     "aws_eks_cluster",
		Name:     r.Name,
		Region:   r.Region,
		Attributes: map[string]any{
			"cloud_id": r.CloudID,
			"name":     r.Name,
			"arn":      r.ARN,
			"version":  r.Version,
			"role":     r.Role,
			"vpc_id":   r.VPCID,
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

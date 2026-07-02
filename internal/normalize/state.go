package normalize

import (
	"fmt"
	"strings"

	"github.com/infancia03/IaC-configuration-drift-detection/internal/model"
	"github.com/infancia03/IaC-configuration-drift-detection/internal/state"
)

// FromState converts Terraform state resources into canonical resources.
func FromState(raw []state.RawResource) []model.Resource {
	out := make([]model.Resource, 0, len(raw))
	for _, r := range raw {
		if mapped, ok := mapStateResource(r); ok {
			out = append(out, mapped)
		}
	}
	return out
}

func mapStateResource(r state.RawResource) (model.Resource, bool) {
	switch r.Type {
	case "aws_s3_bucket":
		return mapAWSS3FromState(r), true
	case "aws_instance":
		return mapAWSInstanceFromState(r), true
	case "aws_iam_role":
		return mapAWSIAMRoleFromState(r), true
	case "aws_security_group":
		return mapAWSSecurityGroupFromState(r), true
	case "aws_db_instance":
		return mapAWSDBInstanceFromState(r), true
	case "aws_lambda_function":
		return mapAWSLambdaFunctionFromState(r), true
	case "aws_eks_cluster":
		return mapAWSEKSClusterFromState(r), true
	default:
		return model.Resource{}, false
	}
}

func mapAWSS3FromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	bucketName := stringAttr(attrs, "bucket")
	region := stringAttr(attrs, "region")
	if region == "" {
		region = "us-east-1"
	}

	attributes := map[string]any{
		"bucket":      bucketName,
		"versioning":  versioningStatus(attrs),
	}

	return model.Resource{
		ID:         canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider:   model.ProviderAWS,
		Type:       r.Type,
		Name:       r.Name,
		Address:    r.Address,
		Region:     region,
		Attributes: attributes,
		Tags:       extractTags(attrs),
	}
}

func mapAWSInstanceFromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	region := defaultRegion(stringAttr(attrs, "region"))
	return model.Resource{
		ID:       canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider: model.ProviderAWS,
		Type:     r.Type,
		Name:     r.Name,
		Address:  r.Address,
		Region:   region,
		Attributes: map[string]any{
			"cloud_id":      stringAttr(attrs, "id"),
			"ami":           stringAttr(attrs, "ami"),
			"instance_type": stringAttr(attrs, "instance_type"),
			"subnet_id":     stringAttr(attrs, "subnet_id"),
			"vpc_id":        stringAttr(attrs, "vpc_id"),
			"private_ip":    stringAttr(attrs, "private_ip"),
		},
		Tags: extractTags(attrs),
	}
}

func mapAWSIAMRoleFromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	region := defaultRegion(stringAttr(attrs, "region"))
	name := stringAttr(attrs, "name")
	if name == "" {
		name = r.Name
	}
	return model.Resource{
		ID:       canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider: model.ProviderAWS,
		Type:     r.Type,
		Name:     r.Name,
		Address:  r.Address,
		Region:   region,
		Attributes: map[string]any{
			"cloud_id":           name,
			"name":               name,
			"arn":                stringAttr(attrs, "arn"),
			"path":               stringAttr(attrs, "path"),
			"assume_role_policy": stringAttr(attrs, "assume_role_policy"),
		},
		Tags: extractTags(attrs),
	}
}

func mapAWSSecurityGroupFromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	region := defaultRegion(stringAttr(attrs, "region"))
	return model.Resource{
		ID:       canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider: model.ProviderAWS,
		Type:     r.Type,
		Name:     r.Name,
		Address:  r.Address,
		Region:   region,
		Attributes: map[string]any{
			"cloud_id":           stringAttr(attrs, "id"),
			"name":               stringAttr(attrs, "name"),
			"description":        stringAttr(attrs, "description"),
			"vpc_id":             stringAttr(attrs, "vpc_id"),
			"ingress_rule_count": listLen(attrs["ingress"]),
			"egress_rule_count":  listLen(attrs["egress"]),
		},
		Tags: extractTags(attrs),
	}
}

func mapAWSDBInstanceFromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	region := defaultRegion(stringAttr(attrs, "region"))
	identifier := stringAttr(attrs, "identifier")
	if identifier == "" {
		identifier = stringAttr(attrs, "id")
	}
	if identifier == "" {
		identifier = r.Name
	}
	return model.Resource{
		ID:       canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider: model.ProviderAWS,
		Type:     r.Type,
		Name:     r.Name,
		Address:  r.Address,
		Region:   region,
		Attributes: map[string]any{
			"cloud_id":            identifier,
			"identifier":          identifier,
			"engine":              stringAttr(attrs, "engine"),
			"instance_class":      stringAttr(attrs, "instance_class"),
			"allocated_storage":   intAttr(attrs, "allocated_storage"),
			"storage_type":        stringAttr(attrs, "storage_type"),
			"multi_az":            boolAttr(attrs, "multi_az"),
			"publicly_accessible": boolAttr(attrs, "publicly_accessible"),
		},
		Tags: extractTags(attrs),
	}
}

func mapAWSLambdaFunctionFromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	region := defaultRegion(stringAttr(attrs, "region"))
	name := stringAttr(attrs, "function_name")
	if name == "" {
		name = r.Name
	}
	return model.Resource{
		ID:       canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider: model.ProviderAWS,
		Type:     r.Type,
		Name:     r.Name,
		Address:  r.Address,
		Region:   region,
		Attributes: map[string]any{
			"cloud_id":      name,
			"function_name": name,
			"arn":           stringAttr(attrs, "arn"),
			"runtime":       stringAttr(attrs, "runtime"),
			"handler":       stringAttr(attrs, "handler"),
			"role":          stringAttr(attrs, "role"),
			"memory_size":   intAttr(attrs, "memory_size"),
			"timeout":       intAttr(attrs, "timeout"),
		},
		Tags: extractTags(attrs),
	}
}

func mapAWSEKSClusterFromState(r state.RawResource) model.Resource {
	attrs := r.Attributes
	region := defaultRegion(stringAttr(attrs, "region"))
	name := stringAttr(attrs, "name")
	if name == "" {
		name = r.Name
	}
	return model.Resource{
		ID:       canonicalID(model.ProviderAWS, r.Type, region, r.Name),
		Provider: model.ProviderAWS,
		Type:     r.Type,
		Name:     r.Name,
		Address:  r.Address,
		Region:   region,
		Attributes: map[string]any{
			"cloud_id": name,
			"name":     name,
			"arn":      stringAttr(attrs, "arn"),
			"version":  stringAttr(attrs, "version"),
			"role":     stringAttr(attrs, "role_arn"),
			"vpc_id":   nestedStringAttr(attrs, "vpc_config", "vpc_id"),
		},
		Tags: extractTags(attrs),
	}
}

func versioningStatus(attrs map[string]any) string {
	raw, ok := attrs["versioning"]
	if !ok {
		return "Disabled"
	}
	switch v := raw.(type) {
	case []any:
		if len(v) == 0 {
			return "Disabled"
		}
		if block, ok := v[0].(map[string]any); ok {
			if enabled, ok := block["enabled"].(bool); ok && enabled {
				return "Enabled"
			}
		}
	case map[string]any:
		if enabled, ok := v["enabled"].(bool); ok && enabled {
			return "Enabled"
		}
	}
	return "Disabled"
}

func extractTags(attrs map[string]any) map[string]string {
	tags := map[string]string{}
	if raw, ok := attrs["tags"].(map[string]any); ok {
		for k, v := range raw {
			tags[k] = fmt.Sprint(v)
		}
	}
	return tags
}

func stringAttr(attrs map[string]any, key string) string {
	if v, ok := attrs[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func intAttr(attrs map[string]any, key string) int {
	switch v := attrs[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func boolAttr(attrs map[string]any, key string) bool {
	v, _ := attrs[key].(bool)
	return v
}

func nestedStringAttr(attrs map[string]any, key, nestedKey string) string {
	switch raw := attrs[key].(type) {
	case []any:
		if len(raw) == 0 {
			return ""
		}
		if block, ok := raw[0].(map[string]any); ok {
			return stringAttr(block, nestedKey)
		}
	case map[string]any:
		return stringAttr(raw, nestedKey)
	}
	return ""
}

func defaultRegion(region string) string {
	if region == "" {
		return "us-east-1"
	}
	return region
}

func listLen(value any) int {
	switch v := value.(type) {
	case []any:
		return len(v)
	case []map[string]any:
		return len(v)
	default:
		return 0
	}
}

func canonicalID(provider model.Provider, typ, region, name string) string {
	return fmt.Sprintf("%s:%s:%s:%s", provider, typ, region, name)
}

// CanonicalBucketID builds the canonical ID used for S3 bucket matching.
func CanonicalBucketID(region, tfName string) string {
	return canonicalID(model.ProviderAWS, "aws_s3_bucket", region, tfName)
}

// MatchNameFromBucket derives the Terraform logical name label from bucket metadata when possible.
func MatchNameFromBucket(tags map[string]string, bucketName string) string {
	if name := tags["Name"]; name != "" {
		return sanitizeName(name)
	}
	parts := strings.Split(bucketName, "-")
	if len(parts) > 0 {
		return sanitizeName(parts[len(parts)-1])
	}
	return sanitizeName(bucketName)
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

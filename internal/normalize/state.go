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

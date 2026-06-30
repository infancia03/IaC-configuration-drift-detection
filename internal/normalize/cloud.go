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
		}
	}
	return out
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

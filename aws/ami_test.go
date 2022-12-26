package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestAmiCompare(t *testing.T) {
	cases := []struct {
		name string
		this *types.Image
		that *types.Image
		ami  string
		ret  bool
	}{
		{
			name: "tags mismatched",
			this: &types.Image{
				Name: aws.String("this"),
				Tags: []types.Tag{
					{
						Key:   aws.String("foo"),
						Value: aws.String("bar"),
					},
				},
			},
			that: &types.Image{
				Name: aws.String("that"),
				Tags: []types.Tag{
					{
						Key:   aws.String("thisFoo"),
						Value: aws.String("bar"),
					},
				},
			},
			ret: false,
		},
		{
			name: "name and tags matched",
			ami:  "this",
			this: &types.Image{
				Name: aws.String("this"),
				Tags: []types.Tag{
					{Key: aws.String("tag1"), Value: aws.String("bar")},
					{Key: aws.String("tag2"), Value: aws.String("bar")},
					{Key: aws.String("tag3"), Value: aws.String("bar")},
				},
			},
			that: &types.Image{
				Name: aws.String("this"),
				Tags: []types.Tag{
					{Key: aws.String("tag1"), Value: aws.String("bar")},
					{Key: aws.String("tag2"), Value: aws.String("bar")},
					{Key: aws.String("tag3"), Value: aws.String("bar")},
				},
			},
			ret: true,
		},
		{
			name: "tags matched but different name",
			ami:  "this",
			this: &types.Image{
				Name: aws.String("this"),
				Tags: []types.Tag{
					{Key: aws.String("tag1"), Value: aws.String("bar")},
					{Key: aws.String("tag2"), Value: aws.String("bar")},
					{Key: aws.String("tag3"), Value: aws.String("bar")},
				},
			},
			that: &types.Image{
				Name: aws.String("that"),
				Tags: []types.Tag{
					{Key: aws.String("tag1"), Value: aws.String("bar")},
					{Key: aws.String("tag2"), Value: aws.String("bar")},
					{Key: aws.String("tag3"), Value: aws.String("bar")},
				},
			},
			ret: false,
		},
		{
			name: "name matched with timestamp",
			ami:  "this-",
			this: &types.Image{
				Name: aws.String("this-2022-08-12T13-02-20Z"),
			},
			that: &types.Image{
				Name: aws.String("this-1900-08-12T13-02-20Z"),
			},
			ret: true,
		},
		{
			name: "name matched with version-timestamp",
			ami:  "this",
			this: &types.Image{
				Name: aws.String("this-v2.41.0-2022-11-07T13-19-36Z"),
			},
			that: &types.Image{
				Name: aws.String("this-v1.0.0-1900-11-07T13-19-36Z"),
			},
			ret: true,
		},
	}

	for _, test := range cases {
		assert.Equal(t, test.ret, amiCompare(test.this, test.that, test.ami), test.name)
	}
}

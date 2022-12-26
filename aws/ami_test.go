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

func TestNextAMIVersion(t *testing.T) {
	cases := []struct {
		name    string
		images  map[string]types.Image
		current types.Image
		ret     string
	}{
		{
			name: "image with semver through labels",
			images: map[string]types.Image{
				"v1": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v1.0.0")}}},
				"v2": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v2.0.0")}}},
			},
			current: types.Image{
				Tags: []types.Tag{
					{
						Key:   aws.String("Version"),
						Value: aws.String("v1.0.0"),
					},
				},
			},
			ret: "v2",
		},
		{
			name: "image without semver",
			images: map[string]types.Image{
				"v1": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("1999-01-01")}}},
				"v2": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("2000-01-01")}}},
			},
			current: types.Image{
				Tags: []types.Tag{
					{
						Key:   aws.String("Version"),
						Value: aws.String("1990-01-01"),
					},
				},
			},
			ret: "v2",
		},
		{
			name:   "empty images",
			images: map[string]types.Image{},
			current: types.Image{
				Name: aws.String("current"),
				Tags: []types.Tag{
					{
						Key:   aws.String("Version"),
						Value: aws.String("1990-01-01"),
					},
				},
			},
			ret: "current",
		},
		{
			name: "images with invalid semver",
			images: map[string]types.Image{
				"v1": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v1.0.0")}}},
				"v2": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v3.0.0")}}},
				"v3": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("2020-01-01")}}},
			},
			current: types.Image{
				Name: aws.String("current"),
				Tags: []types.Tag{
					{
						Key:   aws.String("Version"),
						Value: aws.String("v1.0.0"),
					},
				},
			},
			ret: "v2",
		},
		{
			name: "missing version in tags",
			images: map[string]types.Image{
				"v1": {CreationDate: aws.String("1900-01-01")},
				"v2": {CreationDate: aws.String("2000-01-01")},
			},
			current: types.Image{
				Name:         aws.String("current"),
				CreationDate: aws.String("1900-01-01"),
			},
			ret: "v2",
		},
	}

	// log.SetLevel(log.TraceLevel)
	for _, test := range cases {
		assert.Equal(t, test.ret, nextAMIVersion(test.images, test.current), test.name)
	}
}

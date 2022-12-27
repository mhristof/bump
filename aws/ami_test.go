package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestAmiCompare(t *testing.T) {
	cases := []struct {
		name       string
		this       types.Image
		that       *types.Image
		trimmedAMI string
		ret        bool
	}{
		{
			name: "tags mismatched",
			this: types.Image{
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
			name:       "name and tags matched",
			trimmedAMI: "this",
			this: types.Image{
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
			name:       "tags matched but different name",
			trimmedAMI: "this",
			this: types.Image{
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
			name:       "name matched with timestamp",
			trimmedAMI: "this",
			this: types.Image{
				Name: aws.String("this-2022-08-12T13-02-20Z"),
			},
			that: &types.Image{
				Name: aws.String("this-1900-08-12T13-02-20Z"),
			},
			ret: true,
		},
		{
			name:       "name matched with version-timestamp",
			trimmedAMI: "this",
			this: types.Image{
				Name: aws.String("this-v2.41.0-2022-11-07T13-19-36Z"),
			},
			that: &types.Image{
				Name: aws.String("this-v1.0.0-1900-11-07T13-19-36Z"),
			},
			ret: true,
		},
	}

	// log.SetLevel(log.TraceLevel)
	for _, test := range cases {
		assert.Equal(t, test.ret, amiCompare(test.this, test.that, test.trimmedAMI), test.name)
	}
}

func TestNextAMIVersion(t *testing.T) {
	cases := []struct {
		name    string
		images  map[string]types.Image
		current types.Image
		client  *Client
		ret     string
	}{
		{
			name:   "image with semver through labels",
			client: &Client{AMITags: []string{"Version"}},
			images: map[string]types.Image{
				"v1": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v1.0.0")}}},
				"v2": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v2.0.0")}}},
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
			name:   "image without semver",
			client: &Client{AMITags: []string{"Version"}},
			images: map[string]types.Image{
				"v1": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("1999-01-01")}}},
				"v2": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("2000-01-01")}}},
			},
			current: types.Image{
				Name: aws.String("current"),
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
			client: &Client{AMITags: []string{"Version"}},
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
			name:   "images with invalid semver",
			client: &Client{AMITags: []string{"Version"}},
			images: map[string]types.Image{
				"v1": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v3.0.0")}}},
				"v2": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v4.0.0")}}},
				"v3": {Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("2020-01-01")}}},
			},
			current: types.Image{
				Name: aws.String("current"),
				Tags: []types.Tag{
					{
						Key:   aws.String("Version"),
						Value: aws.String("v3.0.0"),
					},
				},
			},
			ret: "v2",
		},
		{
			name:   "missing version in tags",
			client: &Client{AMITags: []string{""}},
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
		assert.Equal(t, test.ret, nextAMIVersion(test.client, test.images, test.current), test.name)
	}
}

func TestFindAMI(t *testing.T) {
	cases := []struct {
		name     string
		c        *Client
		ami      string
		match    types.Image
		partials []types.Image
	}{
		{
			name: "found exact match without partials",
			ami:  "image1",
			c: &Client{
				accounts: map[string]*Account{
					"account": {
						Images: []types.Image{
							{
								Name: aws.String("image1"),
							},
						},
					},
				},
			},
			match: types.Image{
				Name: aws.String("image1"),
			},
		},
		{
			name: "found exact match with partials",
			ami:  "image1",
			c: &Client{
				accounts: map[string]*Account{
					"account": {
						Images: []types.Image{
							{Name: aws.String("image1")},
							{Name: aws.String("image1-v1.0.0")},
						},
					},
				},
			},
			match: types.Image{
				Name: aws.String("image1"),
			},
			partials: []types.Image{
				{Name: aws.String("image1-v1.0.0")},
			},
		},
		{
			name: "nothing found",
			ami:  "image1",
			c: &Client{
				accounts: map[string]*Account{
					"account": {
						Images: []types.Image{
							{Name: aws.String("image2")},
						},
					},
				},
			},
		},
	}

	for _, test := range cases {
		match, partials := test.c.findAMI(test.ami)
		assert.Equal(t, test.match, match, test.name)
		assert.Equal(t, test.partials, partials, test.name)
	}
}

func TestUpdateAMI(t *testing.T) {
	cases := []struct {
		name string
		c    *Client
		ami  string
		res  string
	}{
		{
			name: "update existing ami",
			ami:  "image40-v1.0.0",
			c: &Client{
				AMITags: []string{"Version"},
				accounts: map[string]*Account{
					"account": {
						Images: []types.Image{
							{Name: aws.String("image40-v1.0.0"), Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v1.0.0")}}},
							{Name: aws.String("image40-v2.0.0"), Tags: []types.Tag{{Key: aws.String("Version"), Value: aws.String("v2.0.0")}}},
						},
					},
				},
			},
			res: "image40-v2.0.0",
		},
		{
			name: "update partial ami",
			ami:  "image-2022.11.30-1669766779",
			c: &Client{
				AMITags: []string{"Release"},
				accounts: map[string]*Account{
					"account": {
						Images: []types.Image{
							{Name: aws.String("image-2022.11.30-9999999990"), Tags: []types.Tag{{Key: aws.String("Release"), Value: aws.String("2022.11.30")}}},
							{Name: aws.String("image-2022.11.31-9999999990"), Tags: []types.Tag{{Key: aws.String("Release"), Value: aws.String("2022.11.31")}}},
						},
					},
				},
			},
			res: "image-2022.11.31-9999999990",
		},
	}

	// log.SetLevel(log.TraceLevel)
	for _, test := range cases {
		assert.Equal(t, test.res, test.c.updateAMI(test.ami), test.name)
	}
}

func TestAMIVersion(t *testing.T) {
	cases := []struct {
		name     string
		image    types.Image
		key      string
		resKey   string
		resValue string
	}{
		{
			name: "creation date without any tags",
			image: types.Image{
				CreationDate: aws.String("2000-01-01"),
			},
			resKey:   "CreationDate",
			resValue: "2000-01-01",
		},
		{
			name: "gitlab ci ref name",
			image: types.Image{
				Tags: []types.Tag{{Key: aws.String("CI_COMMIT_REF_NAME"), Value: aws.String("1")}},
			},
			resKey:   "CI_COMMIT_REF_NAME",
			resValue: "1",
		},
	}

	for _, test := range cases {
		c := Client{AMITags: []string{test.resKey}}

		key, val := c.amiVersion(test.image, test.key)
		assert.Equal(t, test.resKey, key, test.name)
		assert.Equal(t, test.resValue, val, test.name)
	}
}

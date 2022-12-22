package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/stretchr/testify/assert"
)

func TestUpdateECR(t *testing.T) {
	cases := []struct {
		name   string
		repo   string
		client Client
		exp    string
	}{
		{
			name: "single repo without version",
			repo: "foo",
			client: Client{
				"test": &Account{
					ECRImages: []ecrTypes.ImageDetail{
						{
							ImageTags:      []string{"v1.0.0"},
							RepositoryName: aws.String("foo"),
						},
						{
							ImageTags:      []string{"v2.0.0"},
							RepositoryName: aws.String("foo"),
						},
					},
				},
			},
			exp: "foo:v2.0.0",
		},
		{
			name: "unknown repo",
			repo: "foo",
			client: Client{
				"test": &Account{
					ECRImages: []ecrTypes.ImageDetail{},
				},
			},
			exp: "foo",
		},
	}

	for _, test := range cases {
		assert.Equal(t, test.exp, test.client.updateECR(test.repo), test.name)
	}
}

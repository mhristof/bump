package changes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubUpdate(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		newLine string
	}{
		{
			name:    "simple github url",
			line:    "https://github.com/mhristof/bump-semver/releases/download/v0.1.0/semver",
			newLine: "https://github.com/mhristof/bump-semver/releases/download/v0.17.1/semver",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.newLine, githubUpdate(test.line, extractVersion(test.line)), test.name)
		})
	}
}

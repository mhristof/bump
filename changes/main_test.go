package changes

import (
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestGithubUpdate(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		file       string
		newLine    string
		newVersion *semver.Version
	}{
		{
			name:       "simple github url",
			line:       "https://github.com/mhristof/bump-semver/releases/download/v0.1.0/semver",
			newLine:    "https://github.com/mhristof/bump-semver/releases/download/v0.17.1/semver",
			newVersion: semver.MustParse("v0.17.1"),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			line, newVersion := githubUpdate(test.line, extractVersion(test.line))
			assert.Equal(t, test.newLine, line, test.name)
			assert.Equal(t, test.newVersion, newVersion, test.name)
		})
	}
}

func TestParseHCL(t *testing.T) {
	cases := []struct {
		name       string
		module     string
		oldVersion string
		file       string
	}{
		{
			name:       "terraform module",
			module:     "gitlab-runner-1",
			oldVersion: "6.1.2",
			file: generateFile(t, heredoc.Doc(`
				module "gitlab-runner-1" {
				  for_each = var.runners

				  source  = "npalm/gitlab-runner/aws"
				  version = "6.1.2"
				}
			`)),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			changes := parseHCL(test.file)

			assert.Equal(t, test.module, changes[0].Module, test.name)
			assert.Equal(t, test.oldVersion, changes[0].version.String(), test.name)
			assert.NotEqual(t, test.oldVersion, changes[0].newVersion.String(), test.name)
		})
	}
}

func generateFile(t *testing.T, content string) string {
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	return f.Name()
}

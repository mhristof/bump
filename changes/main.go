package changes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/google/go-github/v50/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Changes []*Change

type Format int

const (
	String Format = iota
	Terraform
)

type Change struct {
	line       string
	NewLine    string
	Module     string
	file       string
	version    *semver.Version
	newVersion *semver.Version
	format     Format
}

func (c Change) String() string {
	switch c.format {
	case String:
		return fmt.Sprintf("%s -> %s", c.line, c.NewLine)
	case Terraform:
		return fmt.Sprintf("%s:%s:%s -> %s", c.file, c.Module, c.version, c.newVersion)
	}

	return "unsupportred format"
}

func New(src []string) Changes {
	ret := Changes{}

	for _, s := range src {
		_, err := os.Stat(s)
		if !errors.Is(err, os.ErrNotExist) {
			log.WithField("file", s).Debug("file")

			ret = append(ret, &Change{
				file: s,
			})

			continue
		}

		log.WithField("string", s).Debug("Checking string")

		ver := extractVersion(s)
		if ver != nil {
			log.WithField("string", s).Debug("Found version")
			ret = append(ret, &Change{
				line:    s,
				version: ver,
			})

			continue
		}
	}

	return ret
}

func extractVersion(line string) *semver.Version {
	semverRegex := regexp.MustCompile(`v?\d+\.\d+\.\d+`)

	if semverRegex.MatchString(line) {
		return semver.MustParse(semverRegex.FindString(line))
	}

	return nil
}

func (c *Changes) Update() {
	for _, change := range *c {
		log.WithField("change", change).Debug("Updating")

		switch {
		case strings.HasSuffix(change.file, ".tf"):
			tfChanges := parseHCL(change.file)
			*c = append(*c, tfChanges...)

		case strings.Contains(change.line, "https://gitlab.com"):
			log.WithField("change", change).Debug("Updating gitlab link")

		case strings.Contains(change.line, "https://github.com"):
			log.WithField("change", change).Debug("Updating github link")

			change.NewLine = githubUpdate(change.line, change.version)

			log.WithField("change", change).Debug("Updated github link")
		}
	}

	// remove empty changes.
	ret := Changes{}
	for _, change := range *c {
		if change.NewLine == "" {
			continue
		}
		ret = append(ret, change)
	}

	*c = ret
}

func githubUpdate(line string, version *semver.Version) string {
	ctx := context.Background()

	token := os.Getenv("GITHUB_READONLY_TOKEN")

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	repoRegex := regexp.MustCompile(`https://github.com/([a-zA-Z0-9-]+)/([a-zA-Z0-9-]+)`)

	matches := repoRegex.FindStringSubmatch(line)

	owner := matches[1]
	repo := matches[2]

	// get the repo
	releases, _, _ := client.Repositories.ListReleases(context.TODO(), owner, repo, nil)

	log.WithFields(log.Fields{
		"len":   len(releases),
		"repo":  repo,
		"owner": owner,
	}).Debug("Found releases")
	semverReleases := make([]*semver.Version, len(releases))

	for i, release := range releases {
		log.WithField("release", release.GetTagName()).Debug("Release")
		semverReleases[i] = semver.MustParse(release.GetTagName())
	}

	sort.Sort(semver.Collection(semverReleases))

	for i := len(semverReleases) - 1; i >= 0; i-- {
		if semverReleases[i].Compare(version) > 0 {
			log.WithField("version", semverReleases[i].String()).Debug("Found version")
			return strings.ReplaceAll(line, version.String(), semverReleases[i].String())
		}
	}

	return line
}

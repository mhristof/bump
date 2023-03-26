package changes

import (
	"context"
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

type Change struct {
	line    string
	NewLine string
	version *semver.Version
}

func (c Change) String() string {
	return fmt.Sprintf("%s -> %s", c.line, c.NewLine)
}

func New(src []string) Changes {
	var ret Changes

	for _, s := range src {
		semverRegex := regexp.MustCompile(`v?\d+\.\d+\.\d+`)

		log.WithField("string", s).Debug("Checking string")

		if semverRegex.MatchString(s) {
			log.WithField("string", s).Debug("Found version")
			ret = append(ret, &Change{
				line:    s,
				version: semver.MustParse(semverRegex.FindString(s)),
			})
		}
	}

	return ret
}

func (c *Changes) Update() {
	for _, change := range *c {
		log.WithField("change", change).Debug("Updating")

		switch {
		case strings.Contains(change.line, "https://gitlab.com"):
			log.WithField("change", change).Debug("Updating gitlab link")
		case strings.Contains(change.line, "https://github.com"):
			log.WithField("change", change).Debug("Updating github link")

			change.NewLine = githubUpdate(change.line, change.version)

			log.WithField("change", change).Debug("Updated github link")
		}
	}
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

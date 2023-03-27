package changes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v50/github"
	"github.com/mhristof/bump/awsdata"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Changes []*Change

type Format int

func (f Format) String() string {
	switch f {
	case String:
		return "string"
	case Terraform:
		return "terraform"
	}

	return "unsupported"
}

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
	Source     string
}

func (c Change) String() string {
	switch c.format {
	case String:
		return fmt.Sprintf("%s -- %s -> %s", c.file, c.line, c.NewLine)
	case Terraform:
		return fmt.Sprintf("%s:%s:%s -> %s", c.file, c.Module, c.version, c.newVersion)
	}

	return "unsupportred format"
}

func New(src []string) Changes {
	ret := Changes{}

	for _, s := range src {
		_, err := os.Stat(s)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		log.WithField("file", s).Debug("file")

		ret = append(ret, &Change{
			file: s,
		})

		data, err := os.ReadFile(s)
		if err != nil {
			log.WithField("file", s).Error("Failed to read file")
			continue
		}

		for _, line := range strings.Split(string(data), "\n") {
			ver := extractVersion(line)
			if ver != nil {
				log.WithFields(log.Fields{
					"string":  s,
					"version": ver,
					"line":    line,
				}).Debug("found version")

				ret = append(ret, &Change{
					line:    line,
					version: ver,
					file:    s,
				})

				continue
			}

		}
	}

	for _, s := range src {
		_, err := os.Stat(s)
		if !errors.Is(err, os.ErrNotExist) {
			log.WithField("file", s).Debug("file")

			ret = append(ret, &Change{
				file: s,
			})
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

func (c *Changes) Update(threads int) {
	parsed := map[string]struct{}{}

	aws := awsdata.New(threads)

	var changed Changes

	for _, change := range *c {
		log.WithField("change", change).Trace("checking change")

		switch {
		case strings.Contains(change.line, "dkr.ecr"):
			log.WithFields(log.Fields{
				"change":         change,
				"line":           change.line,
				"change.version": change.version,
			}).Debug("Updating ECR link")

			newContents, newVersion := updateECR(aws, change.line, change.version)
			if newVersion == nil {
				continue
			}
			change.NewLine = newContents
			change.newVersion = newVersion
			changed = append(changed, change)

		case strings.Contains(change.line, "https://gitlab.com"):
			log.WithField("change", change).Debug("Updating gitlab link")

		case strings.Contains(change.line, "https://github.com"):
			log.WithField("change", change).Debug("Updating github link")

			newContents, newVersion := githubUpdate(change.line, change.version)
			change.NewLine = newContents
			change.newVersion = newVersion

			changed = append(changed, change)
			log.WithField("change", change).Debug("Updated github link")

		case strings.HasSuffix(change.file, ".tf"):
			if _, ok := parsed[change.file]; ok {
				log.WithField("file", change.file).Debug("already parsed with HCL")

				continue
			}
			tfChanges := parseHCL(change.file)

			log.WithField("changes", tfChanges).Debug("Found HCL changes")
			parsed[change.file] = struct{}{}
			changed = append(changed, tfChanges...)
		}
	}

	log.WithField("len", len(changed)).Debug("number of changes")

	*c = changed
}

func githubUpdate(line string, version *semver.Version) (string, *semver.Version) {
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
	releases, foo, bar := client.Repositories.ListReleases(context.TODO(), owner, repo, nil)

	log.WithFields(log.Fields{
		"len":   len(releases),
		"repo":  repo,
		"owner": owner,
		"foo":   foo,
		"bar":   bar,
	}).Debug("Found releases")
	semverReleases := make([]*semver.Version, len(releases))

	for i, release := range releases {
		log.WithField("release", release.GetTagName()).Trace("Release")
		semverReleases[i] = semver.MustParse(release.GetTagName())
	}

	sort.Sort(semver.Collection(semverReleases))

	for i := len(semverReleases) - 1; i >= 0; i-- {
		if semverReleases[i].Compare(version) > 0 {
			log.WithField("version", semverReleases[i].String()).Debug("Found version")
			return strings.ReplaceAll(line, version.String(), semverReleases[i].String()), semverReleases[i]
		}
	}

	return line, nil
}

func (c Change) Apply() {
	if c.file == "" {
		return
	}

	data, err := os.ReadFile(c.file)
	if err != nil {
		panic(err)
	}

	switch c.format {
	case String:
		data = []byte(strings.ReplaceAll(string(data), c.line, c.NewLine))
	case Terraform:
		log.WithFields(log.Fields{
			"file":    c.file,
			"version": c.version,
			"new":     c.newVersion,
		}).Debug("Updating terraform file")

		data = []byte(strings.ReplaceAll(string(data), c.version.String(), c.newVersion.String()))
	}

	fileStat, err := os.Stat(c.file)
	if err != nil {
		panic(err)
	}

	os.WriteFile(c.file, data, fileStat.Mode())

	log.WithField("file", c.file).Info("Updated file")
}

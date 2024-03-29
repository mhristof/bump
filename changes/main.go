package changes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

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
	var ret string
	switch c.format {
	case String:
		ret = fmt.Sprintf("%s -- %s -> %s", c.file, c.line, c.NewLine)
	case Terraform:
		ret = fmt.Sprintf("%s:%s:%s -> %s", c.file, c.Module, c.version, c.newVersion)
	}

	if c.Source != "" {
		ret += " " + githubDiffURL(c.Source, c.version.String(), c.newVersion.String())
	}

	return ret
}

func githubDiffURL(repo, from, to string) string {
	urls := []string{
		fmt.Sprintf("%s/compare/%s...%s", repo, from, to),
		fmt.Sprintf("%s/compare/v%s...v%s", repo, from, to),
	}

	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	var ret string

	for _, url := range urls {
		go func(url string) {
			defer wg.Done()

			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == 200 {
				ret = url
			}
		}(url)
	}

	wg.Wait()
	return ret
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

	reDockerhub := regexp.MustCompile(`\w*/\w*:[^\s]*`)

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

		case strings.Contains(change.line, "ghcr.io"):

			re := regexp.MustCompile(`ghcr.io/([^/]+)/([^/]+):(.+)`)
			matches := re.FindStringSubmatch(change.line)

			if len(matches) != 4 {
				log.WithFields(log.Fields{
					"change": change,
					"line":   change.line,
				}).Debug("Failed to parse ghcr.io line")

				continue
			}

			org := matches[1]
			repo := matches[2]
			tag := strings.Trim(matches[3], `"`)

			log.WithFields(log.Fields{
				"change":  change,
				"line":    change.line,
				"matches": matches,
				"org":     org,
				"repo":    repo,
				"tag":     tag,
			}).Debug("Updating ghcr.io link")

			newVersion := githubPackageUpdate(org, repo, semver.MustParse(tag))

			if newVersion == nil {
				log.WithFields(log.Fields{
					"change.line": change.line,
					"tag":         tag,
				}).Debug("Failed to update ghcr.io link")

				continue
			}

			log.WithFields(log.Fields{
				"change.line": change.line,
				"tag":         tag,
				"newVersion":  newVersion,
			}).Debug("Updated ghcr.io link")

			change.NewLine = strings.ReplaceAll(change.line, tag, newVersion.String())

			log.WithFields(log.Fields{
				"change": change,
			}).Debug("Updated ghcr.io link")

			changed = append(changed, change)

		case strings.Contains(change.line, "-ami-"):
			name := strings.Split(change.line, `"`)[1]
			log.WithFields(log.Fields{
				"line": change.line,
				"name": name,
			}).Debug("searching for AMI")

			newAMI := aws.ValidAMI(name)
			if newAMI == "" {
				log.WithFields(log.Fields{
					"line": change.line,
					"name": name,
				}).Trace("no AMI found")

				continue
			}

			change.NewLine = strings.ReplaceAll(change.line, name, newAMI)
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

		case reDockerhub.MatchString(change.line):
			possibleImage := reDockerhub.FindString(change.line)

			log.WithFields(log.Fields{
				"change":        change,
				"possibleImage": possibleImage,
			}).Debug("Updating dockerhub link")

			contents := dockerHub(possibleImage)
			if contents == "" {
				continue
			}

			change.NewLine = strings.ReplaceAll(change.line, possibleImage, contents)
			changed = append(changed, change)
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

	log.WithFields(log.Fields{
		"len": len(semverReleases),
	}).Debug("Sorted releases")

	for i := len(semverReleases) - 1; i >= 0; i-- {
		log.WithFields(log.Fields{
			"version": semverReleases[i].String(),
			"i":       semverReleases[i].String(),
		}).Debug("Checking version")

		if semverReleases[i].Compare(version) > 0 {
			log.WithField("version", semverReleases[i].String()).Debug("Found version")
			return strings.ReplaceAll(line, version.String(), semverReleases[i].String()), semverReleases[i]
		}
	}

	return line, nil
}

func githubPackageUpdate(org, packageName string, version *semver.Version) *semver.Version {
	ctx := context.Background()

	token := os.Getenv("GITHUB_READONLY_TOKEN")

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	for page := 1; ; page++ {
		versions, resp, err := client.Organizations.PackageGetAllVersions(context.Background(), org, "container", packageName, &github.PackageListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 1000,
			},
		})

		log.WithFields(log.Fields{
			"len":           len(versions),
			"org":           org,
			"package":       packageName,
			"resp":          resp,
			"err":           err,
			"resp.LastPage": resp.LastPage,
		}).Debug("found package releases")

		for _, packageVersion := range versions {
			if len(packageVersion.Metadata.Container.Tags) == 0 {
				continue
			}

			for _, tag := range packageVersion.Metadata.Container.Tags {
				ver, err := semver.NewVersion(tag)
				if err != nil {
					log.WithFields(log.Fields{
						"tag":     tag,
						"err":     err,
						"org":     org,
						"package": packageName,
					}).Debug("failed to parse tag")
					continue
				}

				if ver.Compare(version) > 0 {
					log.WithFields(log.Fields{
						"tag":     tag,
						"ver":     ver,
						"package": packageName,
					}).Debug("found version")
					return ver
				}

				if ver.Compare(version) <= 0 {
					return nil
				}

			}
		}
	}

	return nil
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

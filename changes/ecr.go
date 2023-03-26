package changes

import (
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/mhristof/bump/awsdata"
	log "github.com/sirupsen/logrus"
)

func updateECR(data *awsdata.AWS, line string, version *semver.Version) (string, *semver.Version) {
	repoURIRegex := regexp.MustCompile(`(\d*\.dkr\.ecr\..*\.amazonaws\.com\/.*):`)
	repoURI := repoURIRegex.FindStringSubmatch(line)[1]

	versions := data.Tags(repoURI)

	log.WithFields(log.Fields{
		"repoURI":  repoURI,
		"version":  version,
		"versions": versions,
	}).Debug("Versions")

	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].GreaterThan(version) {
			return strings.ReplaceAll(line, version.String(), versions[i].String()), versions[i]
		}
	}

	return line, nil
}

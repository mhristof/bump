package precommit

import (
	"fmt"
	"strings"

	"github.com/mhristof/bump/bash"
	"github.com/mhristof/bump/tool"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Update(path string, dryrun bool) ([]tool.Change, error) {
	stdout, err := bash.Exec(fmt.Sprintf("cd %s && pre-commit autoupdate", path), dryrun)
	if err != nil {
		return []tool.Change{}, errors.Wrap(err, "cannot update pre-commit")
	}

	changes := stdoutToChanges(stdout)

	log.WithFields(log.Fields{
		"stdout":  stdout,
		"changes": changes,
	}).Debug("precommit")

	return changes, nil
}

func stdoutToChanges(stdout string) []tool.Change {
	var changes []tool.Change

	for _, line := range strings.Split(stdout, "\n") {
		// Updating https://github.com/pre-commit/pre-commit-hooks ... updating v4.3.0 -> v4.4.0. string

		if line == "" || strings.Contains(line, "already up to date") {
			continue
		}

		fields := strings.Fields(line)
		if fields[0] != "Updating" {
			continue
		}

		changes = append(changes, tool.Change{
			Repo:       fields[1],
			Version:    strings.Trim(fields[6], "."),
			OldVersion: fields[4],
		})
	}

	return changes
}

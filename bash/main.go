package bash

import (
	"bytes"
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Exec(cmd string, dryrun bool) (string, error) {
	command := exec.Command("bash", "-c", cmd)

	var stdout, stderr bytes.Buffer

	command.Stdout = &stdout
	command.Stderr = &stderr

	log.WithFields(log.Fields{
		"cmd":    cmd,
		"dryrun": dryrun,
	}).Info("executing command")

	if dryrun {
		return "", nil
	}

	err := command.Run()

	outStr, errStr := stdout.String(), stderr.String()
	if err != nil {
		log.WithFields(log.Fields{
			"outStr": outStr,
			"errStr": errStr,
			"cmd":    cmd,
		}).Debug("cannot execute command")

		return "", errors.Wrapf(err, "cannot execute command: %s", cmd)
	}

	return outStr, nil
}

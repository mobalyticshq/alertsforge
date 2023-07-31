package enrichers

import (
	"errors"
	"os/exec"
	"time"

	"github.com/mobalyticshq/alertsforge/sharedtools"
)

type commandEnricher struct {
	alertinfo    AlertInfo
	config       map[string]string
	bucketWriter bucketWriterInterface
}

type CommandEnricherInterface interface {
	Enrich() (map[string]string, error)
}

func NewCommandEnricher(alertinfo AlertInfo, config map[string]string) *commandEnricher {
	return &commandEnricher{alertinfo: alertinfo, config: config, bucketWriter: &bucketWriter{}}
}

func (c *commandEnricher) Enrich() (map[string]string, error) {

	if err := isEnoughConfigParameters(c.config, []string{
		command,
		targetLabelsPrefix,
	}); err != nil {
		return nil, err
	}

	result := map[string]string{}
	command, err := sharedtools.TemplateString(c.config[command], c.alertinfo)
	if err != nil {
		return nil, errors.New("can't parse command")
	}

	cmd := exec.Command("sh", "-c", command)
	stderr := make([]byte, 0)
	stdout, err := cmd.Output()
	if err != nil {
		if erroutput, ok := err.(*exec.ExitError); ok {
			stderr = erroutput.Stderr
		}
	}

	if _, ok := c.config[bucket]; ok {

		filenamePrefix := time.Now().Format("2006-01-02") + "/" + sharedtools.LabelSetToFingerprint(c.alertinfo.Labels) + sharedtools.LabelSetToFingerprint(c.config)
		if len(stdout) > 0 {
			stdOutFilename := filenamePrefix + "_stdout.txt"
			err := c.bucketWriter.writeToBucket(c.config[bucket], stdOutFilename, stdout)
			if err != nil {
				return nil, err
			}
			result[c.config[targetLabelsPrefix]+"_stdout"] = stdOutFilename
		}

		if len(stderr) > 0 {
			stdErrFilename := filenamePrefix + "_stderr.txt"
			err := c.bucketWriter.writeToBucket(c.config[bucket], stdErrFilename, stderr)
			if err != nil {
				return nil, err
			}
			result[c.config[targetLabelsPrefix]+"_stderr"] = stdErrFilename
		}
	} else {
		if len(stdout) > 0 {
			result[c.config[targetLabelsPrefix]+"_stdout"] = string(stdout)
		}
		if len(stderr) > 0 {
			result[c.config[targetLabelsPrefix]+"_stderr"] = string(stderr)
		}
	}
	return result, nil
}

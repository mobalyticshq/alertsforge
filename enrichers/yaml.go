package enrichers

import (
	"github.com/mobalyticshq/alertsforge/sharedtools"
	"gopkg.in/yaml.v3"
)

type yamlEnricher struct {
	alertinfo  AlertInfo
	config     map[string]string
	fileReader fileInterface
}

func NewYamlEnricher(alertinfo AlertInfo, config map[string]string) *yamlEnricher {
	return &yamlEnricher{alertinfo: alertinfo, config: config, fileReader: &fileReader{}}
}
func (y *yamlEnricher) Enrich() (map[string]string, error) {

	if err := isEnoughConfigParameters(y.config, []string{
		fileName,
		targetLabel,
		value,
	}); err != nil {
		return nil, err
	}

	variables := make(map[string]interface{})
	file, err := y.fileReader.getFile(y.config[fileName])

	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(file, variables)
	if err != nil {
		return nil, err
	}
	type extendedAlertInfo struct {
		Labels      map[string]string
		Annotations map[string]string
		Variables   map[string]interface{}
		StartsAt    string
	}

	alertInfoWithVariables := extendedAlertInfo{
		Labels:      y.alertinfo.Labels,
		Annotations: y.alertinfo.Annotations,
		Variables:   variables,
		StartsAt:    y.alertinfo.StartsAt,
	}

	parsedValue, err := sharedtools.TemplateString(y.config[value], alertInfoWithVariables)
	if err != nil {
		return nil, err
	}
	if len(parsedValue) == 0 {
		return nil, nil
	}

	return map[string]string{y.config[targetLabel]: parsedValue}, nil
}

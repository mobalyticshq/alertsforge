package enrichers

import "github.com/mobalyticshq/alertsforge/sharedtools"

type staticEnricher struct {
	alertinfo AlertInfo
	config    map[string]string
}

func NewStaticEnricher(alertinfo AlertInfo, config map[string]string) *staticEnricher {
	return &staticEnricher{alertinfo: alertinfo, config: config}
}

func (s *staticEnricher) Enrich() (map[string]string, error) {
	if err := isEnoughConfigParameters(s.config, []string{
		targetLabel,
		value,
	}); err != nil {
		return nil, err
	}

	parsedValue, err := sharedtools.TemplateString(s.config[value], s.alertinfo)
	if err != nil {
		return nil, err
	}
	if len(parsedValue) == 0 {
		return nil, nil
	}

	return map[string]string{s.config[targetLabel]: parsedValue}, nil
}

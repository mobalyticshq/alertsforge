package enrichers

import (
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mobalyticshq/alertsforge/sharedtools"
)

type prometheusEnricher struct {
	alertinfo AlertInfo
	config    map[string]string
	cli       sharedtools.HTTPInterface
}

func NewPrometheusEnricher(alertinfo AlertInfo, config map[string]string) *prometheusEnricher {
	return &prometheusEnricher{alertinfo: alertinfo, config: config, cli: &sharedtools.HTTPClient{}}
}

func (p *prometheusEnricher) Enrich() (map[string]string, error) {
	newLabels := map[string]string{}
	if err := isEnoughConfigParameters(p.config, []string{
		sourceLabelsPrefix,
		targetLabelsPrefix,
		promql,
		prometheusUrl,
	}); err != nil {
		return nil, err
	}

	query, err := sharedtools.TemplateString(p.config[promql], p.alertinfo)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, p.config[prometheusUrl], nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resBody, err := p.cli.FetchResponse(req)
	if err != nil {
		return nil, err
	}

	jsonparser.ObjectEach(resBody, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if strings.HasPrefix(string(key), p.config[sourceLabelsPrefix]) {
			newLabels[p.config[targetLabelsPrefix]+strings.TrimPrefix(string(key), p.config[sourceLabelsPrefix])] = string(value)
		}
		return nil
	}, "data", "result", "[0]", "metric")

	return newLabels, nil
}

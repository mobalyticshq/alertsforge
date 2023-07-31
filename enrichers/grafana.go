package enrichers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mobalyticshq/alertsforge/sharedtools"
	"go.uber.org/zap"
)

type grafanaEnricher struct {
	alertinfo    AlertInfo
	config       map[string]string
	bucketWriter bucketWriterInterface
	cli          sharedtools.HTTPInterface
}

func NewGrafanaEnricher(alertinfo AlertInfo, config map[string]string) *grafanaEnricher {
	return &grafanaEnricher{alertinfo: alertinfo, config: config, cli: &sharedtools.HTTPClient{}, bucketWriter: &bucketWriter{}}
}

func (e *grafanaEnricher) Enrich() (map[string]string, error) {

	if err := isEnoughConfigParameters(e.config, []string{
		url,
		targetLabel,
		bucket,
	}); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, e.config[url], nil)
	if err != nil {
		zap.S().Info("client: could not create request: %s\n", err)
		return nil, err
	}

	q := req.URL.Query()
	for key, value := range e.config {
		if keyWithoutPrefix, found := strings.CutPrefix(key, "param_"); found { //nolint:typecheck
			if templated, err := sharedtools.TemplateString(value, e.alertinfo); err != nil {
				zap.S().Infof("can't template '%s' with error: %s", value, err)
			} else {
				q.Add(keyWithoutPrefix, templated)
			}

		}
	}

	req.URL.RawQuery = q.Encode()

	zap.S().Debugf("grafana url: %s", req.URL.RawQuery)

	req.Header.Add("Authorization", os.Getenv("AF_GRAFANA_BEARER"))

	resBody, err := e.cli.FetchResponse(req)
	if err != nil {
		return nil, err
	}

	filename := time.Now().Format("2006-01-02") + "/" + sharedtools.LabelSetToFingerprint(e.alertinfo.Labels) + sharedtools.LabelSetToFingerprint(e.config) + ".png"

	e.bucketWriter.writeToBucket(e.config[bucket], filename, resBody)

	return map[string]string{e.config[targetLabel]: filename}, nil
}

package enrichers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

var mockClient = &MockClient{}
var mockFailingClient = &MockFailingClient{}

type MockClient struct {
}
type MockFailingClient struct {
}

func (c *MockClient) FetchResponse(req *http.Request) ([]byte, error) {
	return []byte(`{"data":{"result":[{"metric":{"source_label1":"value1","source_label2":"value2"}},{"metric":{"source_label1":"value3","source_label2":"value4"}}]}}`), nil

}

func (c *MockFailingClient) FetchResponse(req *http.Request) ([]byte, error) {
	return nil, errors.New("can't connect")

}

func TestPrometheusEnricherWithClient(t *testing.T) {
	alertInfo := AlertInfo{
		Labels: map[string]string{},
	}
	enricher := &prometheusEnricher{alertinfo: alertInfo, cli: mockClient}
	t.Run("checking correct work", func(t *testing.T) {
		config := map[string]string{
			sourceLabelsPrefix: "source_",
			targetLabelsPrefix: "target_",
			promql:             `sum(metric_name{alertname="$ALERT_NAME", some_label="$SOME_LABEL"}) by (some_label)`,
			prometheusUrl:      "https://prometheus-url.com",
		}
		enricher.config = config
		newLabels, err := enricher.Enrich()
		require.Nil(t, err)
		expectedNewLabels := map[string]string{
			"target_label1": "value1",
			"target_label2": "value2",
		}
		require.Equal(t, expectedNewLabels, newLabels)
	})

	t.Run("checking error on incorrect template", func(t *testing.T) {
		config := map[string]string{
			sourceLabelsPrefix: "source_",
			targetLabelsPrefix: "target_",
			promql:             `sum(metric_name{{alertname="$ALERT_NAME", some_label="$SOME_LABEL"}) by (some_label)`,
			prometheusUrl:      "https://prometheus-url.com",
		}
		enricher.config = config
		newLabels, err := enricher.Enrich()
		require.Nil(t, newLabels)
		require.Equal(t, err, errors.New("template: value:1: bad character U+003D '='"))
	})

	t.Run("checking error on missing config", func(t *testing.T) {
		config := map[string]string{
			targetLabelsPrefix: "target_",
			promql:             `sum(metric_name{alertname="$ALERT_NAME", some_label="$SOME_LABEL"}) by (some_label)`,
			prometheusUrl:      "https://prometheus-url.com",
		}
		enricher.config = config
		newLabels, err := enricher.Enrich()
		require.Nil(t, newLabels)
		require.Equal(t, err, errors.New("not enough config parameters, 'sourceLabelsPrefix' is mandatory"))
	})

	t.Run("checking fail on connect", func(t *testing.T) {
		config := map[string]string{
			sourceLabelsPrefix: "source_",
			targetLabelsPrefix: "target_",
			promql:             `sum(metric_name{alertname="$ALERT_NAME", some_label="$SOME_LABEL"}) by (some_label)`,
			prometheusUrl:      "https://prometheus-url.com",
		}
		enricher.config = config
		enricher.cli = mockFailingClient
		newLabels, err := enricher.Enrich()
		require.Nil(t, newLabels)
		require.Equal(t, err, errors.New("can't connect"))
	})

	t.Run("checking fail on wrong url", func(t *testing.T) {
		config := map[string]string{
			sourceLabelsPrefix: "source_",
			targetLabelsPrefix: "target_",
			promql:             `sum(metric_name{alertname="$ALERT_NAME", some_label="$SOME_LABEL"}) by (some_label)`,
			prometheusUrl:      string([]byte{0x7f}),
		}
		enricher.config = config
		enricher.cli = mockFailingClient
		newLabels, err := enricher.Enrich()
		require.Nil(t, newLabels)
		require.Equal(t, err.Error(), "parse \"\\x7f\": net/url: invalid control character in URL")
	})
}

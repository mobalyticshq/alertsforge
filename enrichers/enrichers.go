package enrichers

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/mobalyticshq/alertsforge/config"
	"github.com/mobalyticshq/alertsforge/sharedtools"
	"go.uber.org/zap"
)

// Enricher names
const (
	prometheusEnricherName = "prometheus"
	commandEnricherName    = "command"
	breakEnrichername      = "break"
	staticEnricherName     = "static"
	yamlEnricherName       = "yaml"
	grafanaEnricherName    = "grafana"
)

// Possible configuration parameters
const (
	command            = "command"
	url                = "url"
	targetLabel        = "targetLabel"
	bucket             = "bucket"
	promql             = "promql"
	sourceLabelsPrefix = "sourceLabelsPrefix"
	targetLabelsPrefix = "targetLabelsPrefix"
	prometheusUrl      = "prometheusUrl"
	value              = "value"
	fileName           = "fileName"
)

type AlertInfo struct {
	Labels      map[string]string
	Annotations map[string]string
	StartsAt    string
}

type EnrichmentInterface interface {
	StartEnrichmentFlow(alert sharedtools.Alert) []error
}

type EnricherInterface interface {
	Enrich() (map[string]string, error)
}

func NewEnrichment(config *config.RunbooksConfig) EnrichmentInterface {
	return &Enricher{
		config: config,
	}
}

type Enricher struct {
	config *config.RunbooksConfig
}

func (e *Enricher) StartEnrichmentFlow(alert sharedtools.Alert) []error {
	log := zap.S()
	breakEnrichmentFlag := false
	errors := []error{}
	for stepNumber, step := range e.config.EnrichmentFlow {
		log.Debugf("step: %d, %v", stepNumber, step)
		if sharedtools.MatchLabels(alert.Labels, step.LabelsSelector) {
			for runbookNumber, runbook := range step.Runbooks {
				log.Debugf("starting enricher %v", runbook)
				if runbook.EnricherName == breakEnrichername {
					breakEnrichmentFlag = true
					break
				}
				newlabels, err := startEnricher(runbook, alert)
				if err != nil {
					log.Error(err)
					sharedtools.MergeMaps(alert.Labels, map[string]string{fmt.Sprintf("alertsforge_errors_step%d_runbook%d", stepNumber, runbookNumber): err.Error()})
					errors = append(errors, fmt.Errorf("alertsforge_errors_step%d_runbook%d", stepNumber, runbookNumber))
				}
				if len(newlabels) > 0 {
					log.Debugf("enriching labels with: %v", newlabels)
					sharedtools.MergeMaps(alert.Labels, newlabels)
				}

			}
			if breakEnrichmentFlag {
				break
			}

		} else {
			log.Debugf("step %d skipped", stepNumber)
		}

	}

	log.Debugf("resulting labels map: %v", alert.Labels)
	return errors
}

func startEnricher(runbook config.Runbook, alert sharedtools.Alert) (map[string]string, error) {
	alertinfo := AlertInfo{
		Labels:      alert.Labels,
		Annotations: alert.Annotations,
		StartsAt:    alert.StartsAt.String(),
	}
	var enricher EnricherInterface

	switch runbook.EnricherName {
	case prometheusEnricherName:
		enricher = NewPrometheusEnricher(alertinfo, runbook.Config)
	case commandEnricherName:
		enricher = NewCommandEnricher(alertinfo, runbook.Config)
	case staticEnricherName:
		enricher = NewStaticEnricher(alertinfo, runbook.Config)
	case yamlEnricherName:
		enricher = NewYamlEnricher(alertinfo, runbook.Config)
	case grafanaEnricherName:
		enricher = NewGrafanaEnricher(alertinfo, runbook.Config)
	}

	if enricher != nil {
		return enricher.Enrich()
	}
	return nil, errors.New("enricher " + runbook.EnricherName + " not found")

}

func isEnoughConfigParameters(config map[string]string, mandatory []string) error {
	for _, parameter := range mandatory {
		_, ok := config[parameter]
		if !ok {
			return errors.New("not enough config parameters, '" + parameter + "' is mandatory")
		}
	}
	return nil
}

func getBucketWriter(ctx context.Context, bucket, filename string) (*storage.Writer, error) {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	storagebucket := storageClient.Bucket(bucket)
	object := storagebucket.Object(filename)
	w := object.NewWriter(ctx)
	return w, nil
}

type bucketWriter struct{}
type bucketWriterInterface interface {
	writeToBucket(bucketName, stdOutFilename string, data []byte) error
}

func (b *bucketWriter) writeToBucket(bucketName, stdOutFilename string, data []byte) error {
	ctx := context.Background()
	w, err := getBucketWriter(ctx, bucketName, stdOutFilename)
	if err != nil {
		return err
	}
	defer w.Close()
	w.Write(data)
	return nil
}

type fileInterface interface {
	getFile(filePath string) ([]byte, error)
}

type fileReader struct{}

func (c *fileReader) getFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

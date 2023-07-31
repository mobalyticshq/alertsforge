package alertsink

import (
	"github.com/mobalyticshq/alertsforge/config"
	"github.com/mobalyticshq/alertsforge/sharedtools"
)

type SinkInterface interface {
	SendAlerts(alerts []sharedtools.Alert) (accepted []string, resolved []string, errors []error)
}

const (
	Oncall = "oncall"
)

func NewAlertSink(sinkName string, runbooks *config.RunbooksConfig) SinkInterface {
	return NewOncallSink(runbooks)
}

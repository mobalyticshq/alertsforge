package alertsource

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"net/http"

	"github.com/mobalyticshq/alertsforge/alertsink"
	"github.com/mobalyticshq/alertsforge/config"
	"github.com/mobalyticshq/alertsforge/enrichers"
	"github.com/mobalyticshq/alertsforge/sharedtools"
	"go.uber.org/zap"
)

type AlertManager struct {
	AlertsBuffer     map[string]*sharedtools.Alert
	AlertBufferMutex sync.RWMutex
	AlertSink        alertsink.SinkInterface
	AlertEnricher    enrichers.EnrichmentInterface
	runbooks         *config.RunbooksConfig
}
type AlertManagerInterface interface {
	AlertsProcessor()
	ProcessAlertsBuffer() []error
	receiveAlerts(alerts []sharedtools.Alert)
	ProcessAlertsBufferWebhook(w http.ResponseWriter, r *http.Request)
	ShowAlertsBufferWebhook(w http.ResponseWriter, r *http.Request)
	AlertWebhook(w http.ResponseWriter, r *http.Request)
}

func NewAlertManager(runbooks *config.RunbooksConfig) AlertManagerInterface {

	return &AlertManager{
		AlertsBuffer:     map[string]*sharedtools.Alert{},
		AlertBufferMutex: sync.RWMutex{},
		runbooks:         runbooks,
		AlertSink:        alertsink.NewAlertSink(alertsink.Oncall, runbooks),
		AlertEnricher:    enrichers.NewEnrichment(runbooks),
	}
}

func (a *AlertManager) AlertsProcessor() {
	for range time.Tick(time.Second * 10) {
		errs := a.ProcessAlertsBuffer()
		if len(errs) > 0 {
			zap.S().Warnf("buffer processsing was not coompletely successful")
		}
	}
}

func (a *AlertManager) ProcessAlertsBufferWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log := zap.S()
	body := ""
	if err := a.ProcessAlertsBuffer(); err != nil {
		body = fmt.Sprintf("%s", err)
		log.Errorf("can't process alerts buffer", err)
	} else {
		body = "success"
	}
	asJson(w, http.StatusOK, body)
}

func (a *AlertManager) ShowAlertsBufferWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	a.AlertBufferMutex.Lock()
	bytes, _ := json.MarshalIndent(a.AlertsBuffer, "", "\t")
	a.AlertBufferMutex.Unlock()
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(bytes))
}

func (a *AlertManager) ProcessAlertsBuffer() []error {
	log := zap.S()
	log.Debugf("starting processing of alertsbuffer")
	AlertsBufferCopy := map[string]sharedtools.Alert{}
	a.AlertBufferMutex.Lock()
	for fingerprint, alert := range a.AlertsBuffer {
		AlertsBufferCopy[fingerprint] = sharedtools.CopyAlert(alert)
	}
	a.AlertBufferMutex.Unlock()

	var wg sync.WaitGroup
	sentAlerts := 0
	errChan := make(chan []error, len(AlertsBufferCopy))
	alertsToOncall := []sharedtools.Alert{}
	alertsToOncallMutex := &sync.RWMutex{}
	log.Debugf("found %d alerts in buffer", len(AlertsBufferCopy))

	for _, alert := range AlertsBufferCopy {
		alertCopy := sharedtools.CopyAlert(&alert)
		if alertCopy.EndsAt.Before(time.Now()) {
			log.Infof("alert end date before current time, consider it resolved: %v", alertCopy)
			if alertCopy.Status == sharedtools.Pending {
				log.Warnf("alert was removed before sending it to oncall! : %v", alertCopy)
				a.AlertBufferMutex.Lock()
				delete(a.AlertsBuffer, alertCopy.Fingerprint)
				a.AlertBufferMutex.Unlock()
			} else {
				sentAlerts++
				wg.Add(1)
				go func() {
					defer wg.Done()
					log.Infof("resolving alert: %v", alertCopy)
					alertCopy.Status = sharedtools.Resolved
					alertsToOncallMutex.Lock()
					alertsToOncall = append(alertsToOncall, alertCopy)
					alertsToOncallMutex.Unlock()
				}()
			}

			continue
		}

		if alertCopy.Status == sharedtools.Pending {
			sentAlerts++
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Infof("found pending alert, enriching it and sending to oncall: %v", alertCopy)
				errs := a.AlertEnricher.StartEnrichmentFlow(alertCopy)
				errChan <- errs
				a.AlertBufferMutex.Lock()
				a.AlertsBuffer[alertCopy.Fingerprint] = &alertCopy
				a.AlertBufferMutex.Unlock()
				alertsToOncallMutex.Lock()
				alertsToOncall = append(alertsToOncall, alertCopy)
				alertsToOncallMutex.Unlock()
			}()

		}

		if resink, err := time.ParseDuration(os.Getenv("AF_RESINK_TIME")); err == nil {
			if alertCopy.Status == sharedtools.Firing && time.Since(alertCopy.LastSinkAt) > resink {
				alertsToOncallMutex.Lock()
				alertsToOncall = append(alertsToOncall, alertCopy)
				alertsToOncallMutex.Unlock()
			}
		}

	}

	wg.Wait()
	close(errChan)

	for er := range errChan {
		for err := range er {
			log.Errorf("Got error from enrichment: %v", err)
		}
	}

	errors := []error{}
	if sentAlerts > 0 {
		log.Infof("alerts to oncall: %v", alertsToOncall)
		var accepted, resolved []string
		accepted, resolved, errors = a.AlertSink.SendAlerts(alertsToOncall)

		log.Infof("accepted fingerprints: %v", accepted)
		log.Infof("resolved fingerprints: %v", resolved)

		for _, err := range errors {
			log.Errorf("error while sending alert to oncall: %s", err)
		}

		a.AlertBufferMutex.Lock()
		for _, fingerprint := range accepted {

			if alert, ok := a.AlertsBuffer[fingerprint]; ok {
				if alert.Status == sharedtools.Firing {
					log.Infof("fingerprint %s have been resinked", alert.Fingerprint)
				} else {
					log.Infof("changing fingerprint %s with status %s to status firing", alert.Fingerprint, alert.Status)
					alert.Status = sharedtools.Firing
				}
				alert.LastSinkAt = time.Now()
			}
		}
		for _, fingerprint := range resolved {
			log.Infof("deleting alert with fingerprint %s from buffer", fingerprint)
			delete(a.AlertsBuffer, fingerprint)
		}
		a.AlertBufferMutex.Unlock()
		log.Infof("%d alerts have been sent to Oncall successfully", sentAlerts)

	}
	log.Debugf("finished processing of alertsbuffer")
	return errors
}

func (a *AlertManager) AlertWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log := zap.S()
	alerts := []sharedtools.Alert{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Can't get body", err)
		return
	}

	if err := json.Unmarshal(body, &alerts); err != nil {
		asJson(w, http.StatusBadRequest, err.Error())
		log.Errorf("Can't unmarshal request from alertmanager, body: \n%s", body)
		return
	}

	a.receiveAlerts(alerts)

	log.Debugf("got %d firing alerts from alertsource", len(alerts))
	asJson(w, http.StatusOK, "success")
}

func (a *AlertManager) receiveAlerts(alerts []sharedtools.Alert) {
	log := zap.S()
	for _, alert := range alerts {
		silenced := false
		alert := alert
		for _, silence := range a.runbooks.Silences {
			if sharedtools.MatchLabels(alert.Labels, silence.LabelsSelector) {
				silenced = true
				break
			}
		}
		if !silenced {
			alert.LastReceiveAt = time.Now()
			copyLabels := sharedtools.CopyMap(alert.Labels)
			delete(copyLabels, "uid") //reduce duplication of alerts
			alert.Fingerprint = sharedtools.LabelSetToFingerprint(copyLabels)

			a.AlertBufferMutex.Lock()
			if alertsforge_delay_resolve, ok := alert.Labels["alertsforge_delay_resolve"]; ok {
				delayDuration, err := time.ParseDuration(alertsforge_delay_resolve)
				if err != nil {
					log.Errorf("can't parse delay duration: %s", err)
				} else {
					alert.Labels["alertsForge_delayed_resolve"] = alert.EndsAt.Add(delayDuration).String()
					alert.EndsAt = alert.EndsAt.Add(delayDuration)
				}
			} else if delayDuration, err := time.ParseDuration(os.Getenv("AF_DEFAULT_RESOLVE_DELAY")); err == nil {
				alert.Labels["alertsForge_delayed_resolve"] = delayDuration.String()
				alert.EndsAt = alert.EndsAt.Add(delayDuration)
			}
			if _, ok := a.AlertsBuffer[alert.Fingerprint]; ok {
				(a.AlertsBuffer[alert.Fingerprint]).EndsAt = alert.EndsAt
				(a.AlertsBuffer[alert.Fingerprint]).LastReceiveAt = time.Now()
			} else {
				alert.Status = sharedtools.Pending
				a.AlertsBuffer[alert.Fingerprint] = &alert
			}
			a.AlertBufferMutex.Unlock()
		}
	}
}

func asJson(w http.ResponseWriter, status int, message string) {
	type responseJSON struct {
		Status  int
		Message string
	}
	data := responseJSON{
		Status:  status,
		Message: message,
	}
	bytes, _ := json.Marshal(data)
	json := string(bytes[:])

	w.WriteHeader(status)
	fmt.Fprint(w, json)
}

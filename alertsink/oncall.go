package alertsink

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/buger/jsonparser"
	"github.com/mobalyticshq/alertsforge/config"
	"github.com/mobalyticshq/alertsforge/sharedtools"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

type OncallRequest struct {
	Title                    string              `json:"title,omitempty"`
	State                    string              `json:"state,omitempty"`
	SlackMessage             string              `json:"slack_message,omitempty"`
	WebMessage               string              `json:"web_message,omitempty"`
	SimpleMessage            string              `json:"simple_message,omitempty"`
	TelegramMessage          string              `json:"telegram_message,omitempty"`
	EscalationChain          string              `json:"escalation_chain,omitempty"`
	AlertmanagerOriginAlerts []sharedtools.Alert `json:"alertmanager_messages,omitempty"`
}

type OncallGetterInterface interface {
	getAlertgroups(groups *[][]byte, state string, page string, pagenum int) error
	getActiveAlertgroups() ([][]byte, error)
	getAlertgroupAlertsByGroupID(alertGroupID string) ([][]byte, error)
}

type OncallSetterInterface interface {
	doOncallIncident(oncall OncallRequest) error
}

type OncallSink struct {
	runbooks  *config.RunbooksConfig
	oncallGet OncallGetterInterface
	oncallSet OncallSetterInterface
}

type OncallGetter struct {
}

type OncallSetter struct {
}

func NewOncallSink(runbooks *config.RunbooksConfig) *OncallSink {
	return &OncallSink{runbooks: runbooks, oncallGet: &OncallGetter{}, oncallSet: &OncallSetter{}}
}

func (o *OncallSetter) doOncallIncident(oncall OncallRequest) error {

	jsonBody, _ := json.Marshal(oncall)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, os.Getenv("AF_ONCALL_API_URL")+"/integrations/v1/formatted_webhook/<place>/", bodyReader)
	if err != nil {
		zap.S().Errorf("client: could not create request: %s\n", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		zap.S().Errorf("client: error making http request: %s\n", err)
		return err
	}
	defer res.Body.Close()
	// log.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zap.S().Errorf("client: could not read response body: %s\n", err)
		return err
	}
	zap.S().Debugf("client: response body: %s\n", resBody)

	return nil
}

func (o *OncallGetter) getActiveAlertgroups() ([][]byte, error) {
	var groups [][]byte
	alertgroupURL := os.Getenv("AF_ONCALL_API_URL") + "/api/v1/alert_groups/"
	if err := o.getAlertgroups(&groups, "new", alertgroupURL, 0); err != nil {
		return nil, err
	}
	if err := o.getAlertgroups(&groups, "acknowledged", alertgroupURL, 0); err != nil {
		return nil, err
	}
	return groups, nil
}

func (o *OncallGetter) getAlertgroups(groups *[][]byte, state string, page string, pagenum int) error {
	if pagenum > 50 {
		return errors.New("too many pages in result")
	}

	req, err := http.NewRequest(http.MethodGet, page, nil)
	if err != nil {
		zap.S().Info("client: could not create request: %s\n", err)
		return err
	}
	if pagenum == 0 {
		q := req.URL.Query()
		q.Add("state", state)
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", os.Getenv("AF_ONCALL_BEARER"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		zap.S().Errorf("client: error making http request: %s\n", err)
		return err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zap.S().Errorf("client: could not read response body: %s\n", err)
		return err
	}
	alertgroups, _, _, agerr := jsonparser.Get(resBody, "results")
	if agerr != nil {
		zap.S().Errorf("could not unmarshal json: %s", agerr)
		return agerr
	}

	if len(alertgroups) > 0 {
		*groups = append(*groups, alertgroups)
	}

	next, nexterr := jsonparser.GetString(resBody, "next")
	if nexterr != nil {
		return nil //nolint:nilerr
		// When next page does not exists means we are on the last page
	}

	if next != "" {
		pagenum++
		err := o.getAlertgroups(groups, state, next, pagenum)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *OncallGetter) getAlertgroupAlertsByGroupID(alertGroupID string) ([][]byte, error) {
	var alerts [][]byte
	err := o.getAlertgroupDetails(&alerts, alertGroupID, os.Getenv("AF_ONCALL_API_URL")+"/api/v1/alerts/", 0)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func (o *OncallGetter) getAlertgroupDetails(alerts *[][]byte, alertGroupID string, page string, pagenum int) error {
	if pagenum > 50 {
		return errors.New("too many pages in result")
	}

	req, err := http.NewRequest(http.MethodGet, page, nil)
	if err != nil {
		log.Printf("client: could not create request: %s\n", err)
		return err
	}
	if pagenum == 0 {
		q := req.URL.Query()
		q.Add("alert_group_id", alertGroupID)
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", os.Getenv("AF_ONCALL_BEARER"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		zap.S().Errorf("client: error making http request: %s\n", err)
		return err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zap.S().Errorf("client: could not read response body: %s\n", err)
		return err
	}
	resalerts, _, _, alerr := jsonparser.Get(resBody, "results")
	if alerr != nil {
		zap.S().Errorf("could not unmarshal json: %s", alerr)
		return alerr
	}

	if len(resalerts) > 0 {
		*alerts = append(*alerts, resalerts)
	}

	next, nexterr := jsonparser.GetString(resBody, "next")
	if nexterr != nil {
		return nexterr
	}

	if next != "" {
		pagenum++
		err := o.getAlertgroups(alerts, alertGroupID, next, pagenum)
		if err != nil {
			return err
		}
	}
	return nil

}

func (o OncallSink) SendAlerts(alerts []sharedtools.Alert) (accepted []string, resolved []string, errors []error) {
	log := zap.L().Sugar()
	groupedAlerts := map[string][]sharedtools.Alert{}

	for _, alert := range alerts {

		variables := AlertTemplate{
			Labels:      alert.Labels,
			Annotations: alert.Annotations,
		}
		title := sharedtools.MustTemplateString(o.runbooks.OncallMessage.Title, variables, "error while parsing title")
		if _, ok := groupedAlerts[title]; !ok {
			groupedAlerts[title] = []sharedtools.Alert{alert}
		} else {
			groupedAlerts[title] = append(groupedAlerts[title], alert)
		}

	}

	alertgroupsInOncall, err := o.oncallGet.getActiveAlertgroups()
	if err != nil {
		log.Errorf("can't get alertgroups: %s", err)
		return
	}

	for title, alertsInGroup := range groupedAlerts {
		oncall := OncallRequest{}
		var latestOncallAlert []byte
		oncall.Title = title

		for _, oncallAlertgroup := range alertgroupsInOncall {
			_, err := jsonparser.ArrayEach(oncallAlertgroup, func(alertgroup []byte, dataType jsonparser.ValueType, offset int, eachErr error) {
				agtitle, err := jsonparser.GetString(alertgroup, "title")
				if err != nil {
					log.Warnf("can't get title of alertgroup: %s", err)
				}
				if agtitle == oncall.Title {

					agid, err := jsonparser.GetString(alertgroup, "id")
					if err != nil {
						log.Warnf("can't get id: %s", err)
					}
					agDetails, err := o.oncallGet.getAlertgroupAlertsByGroupID(agid)
					if err == nil && len(agDetails) > 0 {
						_, err := jsonparser.ArrayEach(agDetails[len(agDetails)-1], func(value []byte, dataType jsonparser.ValueType, offset int, eachErr error) {
							latestOncallAlert = value
						})
						if err != nil {
							log.Warnf("can't iterate alertgroup details: %s", err)
						}

					}

				}
			})
			if err != nil {
				log.Warnf("can't iterate alertgroup: %s", err)
			}

		}

		if len(latestOncallAlert) > 0 {
			latestAlertPayload, _, _, err := jsonparser.Get(latestOncallAlert, "payload")
			if err != nil {
				log.Warnf("can't get latest alert payload: %s", err)
			} else {
				latestOncallAlertObject := OncallRequest{}
				if len(latestOncallAlert) > 0 {
					err := json.Unmarshal(latestAlertPayload, &latestOncallAlertObject)
					if err != nil {
						log.Warnf("can't unmarshal latest oncall alert: %s", err)
					} else if len(latestOncallAlertObject.AlertmanagerOriginAlerts) > 0 {
						oncall.AlertmanagerOriginAlerts = latestOncallAlertObject.AlertmanagerOriginAlerts
					}

				}
			}
		}

		if acceptedInGroup, resolvedInGroup, ok := o.prepareOncallMessage(&oncall, alertsInGroup); ok {

			if err := o.oncallSet.doOncallIncident(oncall); err != nil {
				log.Errorf("Can't create oncall incident: \n%s", err.Error())
				errors = append(errors, err)
			} else {
				accepted = append(accepted, acceptedInGroup...)
				resolved = append(resolved, resolvedInGroup...)
			}
		}
	}

	return
}

type AlertTemplate struct {
	Labels         map[string]string
	Annotations    map[string]string
	FiringAlerts   []sharedtools.Alert
	ResolvedAlerts []sharedtools.Alert
}

func (o OncallSink) prepareOncallMessage(
	oncallRequest *OncallRequest,
	newalerts []sharedtools.Alert,
) (
	acceptedFingerprints []string,
	resolvedFingerprints []string,
	success bool,
) {
	acceptedFingerprints = []string{}
	resolvedFingerprints = []string{}
	resolvedAlerts := map[string]sharedtools.Alert{}
	firingAlerts := map[string]sharedtools.Alert{}
	addToExistingAlerts := []sharedtools.Alert{}
	alertgroupHasUnresolvedAlerts := false

	for _, newalert := range newalerts {
		alertExists := false
		if newalert.Status == sharedtools.Pending {
			newalert.Status = sharedtools.Firing
		}
		for i, alertAM := range oncallRequest.AlertmanagerOriginAlerts {

			if alertAM.Fingerprint == newalert.Fingerprint {
				zap.S().Infof("there is already existing alert with fingerprint: %s", alertAM.Fingerprint)
				alertExists = true
				if newalert.Status == sharedtools.Resolved {
					resolvedFingerprints = append(resolvedFingerprints, newalert.Fingerprint)
				}
				if newalert.Status == sharedtools.Firing {
					acceptedFingerprints = append(acceptedFingerprints, newalert.Fingerprint)
				}

				if alertAM.Status == newalert.Status {
					zap.S().Infof("skipping alert as it's fingerprint already exists with same status in alertgroup, title: %s", oncallRequest.Title)
				}

				zap.S().Infof("incoming alert status: %s, existing alert status: %s", newalert.Status, alertAM.Status)
				alertAM.Status = newalert.Status
				oncallRequest.AlertmanagerOriginAlerts[i].Status = newalert.Status
				oncallRequest.AlertmanagerOriginAlerts[i].EndsAt = newalert.EndsAt
			}

		}

		if !alertExists {
			if newalert.Status == sharedtools.Resolved {
				resolvedFingerprints = append(resolvedFingerprints, newalert.Fingerprint)
			} else {
				acceptedFingerprints = append(acceptedFingerprints, newalert.Fingerprint)
			}
			addToExistingAlerts = append(addToExistingAlerts, newalert)
		}

	}
	oncallRequest.AlertmanagerOriginAlerts = append(oncallRequest.AlertmanagerOriginAlerts, addToExistingAlerts...)

	for _, alertAM := range oncallRequest.AlertmanagerOriginAlerts {

		if alertAM.Status == sharedtools.Resolved {
			resolvedAlerts[alertAM.Fingerprint] = alertAM
		} else {
			alertgroupHasUnresolvedAlerts = true
			firingAlerts[alertAM.Fingerprint] = alertAM
		}
	}

	var firingAlertsSlice sharedtools.AlertsSlice
	var resolvedAlertsSlice sharedtools.AlertsSlice
	firingAlertsSlice = maps.Values(firingAlerts)
	resolvedAlertsSlice = maps.Values(resolvedAlerts)
	sort.Sort(firingAlertsSlice)
	sort.Sort(resolvedAlertsSlice)

	variables := AlertTemplate{
		FiringAlerts:   firingAlertsSlice,
		ResolvedAlerts: resolvedAlertsSlice,
	}

	oncallRequest.WebMessage = sharedtools.MustTemplateString(o.runbooks.OncallMessage.WebMessage, variables, "error while parsing web message")
	oncallRequest.TelegramMessage = sharedtools.MustTemplateString(o.runbooks.OncallMessage.TelegramMessage, variables, "error while parsing telegram message")
	oncallRequest.SlackMessage = sharedtools.MustTemplateString(o.runbooks.OncallMessage.SlackMessage, variables, "error while parsing slack message")
	oncallRequest.SimpleMessage = sharedtools.MustTemplateString(o.runbooks.OncallMessage.SimpleMessage, variables, "error while parsing simple message")
	oncallRequest.EscalationChain = sharedtools.MustTemplateString(o.runbooks.OncallMessage.EscalationChain, variables, "error while parsing simple message")
	if alertgroupHasUnresolvedAlerts {
		oncallRequest.State = sharedtools.Firing
	} else {
		zap.S().Infof("all alerts in alertgroup are in resolved state, alertgroup can be resolved, title: %s", oncallRequest.Title)
		oncallRequest.State = "ok"
	}

	success = true
	return
}

package alertsink

import (
	"testing"
	"time"

	"github.com/mobalyticshq/alertsforge/config"
	"github.com/mobalyticshq/alertsforge/sharedtools"
	"github.com/stretchr/testify/assert"
)

var template = `{{- $last_commits := list }}
{{- range .FiringAlerts }}
{{ .Annotations.description }}
{{- end }}
{{- if .ResolvedAlerts }}
Resolved:
{{- range .ResolvedAlerts }}
{{.Annotations.description}}
{{- end}}
{{- end}}`

var runbooks = config.RunbooksConfig{

	OncallMessage: config.OncallMessage{
		WebMessage:      template,
		TelegramMessage: template,
		SlackMessage:    template,
		SimpleMessage:   template,
	},
}

func TestPrepareOncallMessage_Firing(t *testing.T) {

	hour, _ := time.ParseDuration("1h")

	var oncallRequest = OncallRequest{
		Title: "Test Alert Group",
		AlertmanagerOriginAlerts: []sharedtools.Alert{
			{
				Fingerprint: "1",
				Annotations: map[string]string{"description": "1"},
				Status:      sharedtools.Resolved,
				StartsAt:    time.Now(),
			},
			{
				Fingerprint: "2",
				Annotations: map[string]string{"description": "2"},
				Status:      sharedtools.Firing,
				StartsAt:    time.Now().Add(hour * 2),
			},
			{
				Fingerprint: "3",
				Annotations: map[string]string{"description": "3"},
				Status:      sharedtools.Firing,
				StartsAt:    time.Now().Add(hour * 3),
			},
		},
	}

	newAlerts := []sharedtools.Alert{
		{
			Fingerprint: "1",
			Annotations: map[string]string{"description": "1"},
			Status:      sharedtools.Resolved,
			StartsAt:    time.Now().Add(hour),
		},
		{
			Fingerprint: "2",
			Annotations: map[string]string{"description": "2"},
			Status:      sharedtools.Resolved,
			StartsAt:    time.Now().Add(hour * 2),
		},
		{
			Fingerprint: "3",
			Status:      sharedtools.Firing,
			Annotations: map[string]string{"description": "3"},
			StartsAt:    time.Now().Add(hour * 3),
		},
		{
			Fingerprint: "4",
			Status:      sharedtools.Pending,
			Annotations: map[string]string{"description": "4"},
			StartsAt:    time.Now().Add(hour * 4),
		},
	}

	oncall := OncallSink{runbooks: &runbooks}
	acceptedFingerprints, resolvedFingerprints, success := oncall.prepareOncallMessage(&oncallRequest, newAlerts)

	// Check the expected outputs
	assert.True(t, success)
	assert.Equal(t, []string{"3", "4"}, acceptedFingerprints)
	assert.Equal(t, []string{"1", "2"}, resolvedFingerprints)
	assert.Equal(t, sharedtools.Firing, oncallRequest.State)
	assert.Equal(t, "\n4\n3\nResolved:\n2\n1", oncallRequest.WebMessage)
}

func TestPrepareOncallMessage_Resolved(t *testing.T) {
	hour, _ := time.ParseDuration("1h")
	var oncallRequest = OncallRequest{
		Title: "Test Alert Group",

		AlertmanagerOriginAlerts: []sharedtools.Alert{
			{
				Fingerprint: "1",
				Annotations: map[string]string{"description": "1"},
				Status:      sharedtools.Resolved,
				StartsAt:    time.Now().Add(hour),
			},
			{
				Fingerprint: "2",
				Annotations: map[string]string{"description": "2"},
				Status:      sharedtools.Firing,
				StartsAt:    time.Now().Add(hour * 2),
			},
			{
				Fingerprint: "3",
				Annotations: map[string]string{"description": "3"},
				Status:      sharedtools.Firing,
				StartsAt:    time.Now().Add(hour * 3),
			},
		},
	}

	newAlerts := []sharedtools.Alert{
		{
			Fingerprint: "2",
			Annotations: map[string]string{"description": "2"},
			Status:      sharedtools.Resolved,
			StartsAt:    time.Now().Add(hour * 2),
		},
		{
			Fingerprint: "3",
			Status:      sharedtools.Resolved,
			Annotations: map[string]string{"description": "3"},
			StartsAt:    time.Now().Add(hour * 3),
		},
		{
			Fingerprint: "4",
			Status:      sharedtools.Resolved,
			Annotations: map[string]string{"description": "4"},
			StartsAt:    time.Now().Add(hour * 4),
		},
	}

	oncall := OncallSink{runbooks: &runbooks}
	acceptedFingerprints, resolvedFingerprints, success := oncall.prepareOncallMessage(&oncallRequest, newAlerts)

	// Check the expected outputs
	assert.True(t, success)
	assert.Equal(t, []string{}, acceptedFingerprints)
	assert.Equal(t, []string{"2", "3", "4"}, resolvedFingerprints)
	assert.Equal(t, "ok", oncallRequest.State)
	assert.Equal(t, "\nResolved:\n4\n3\n2\n1", oncallRequest.WebMessage)

}

type OncallGetTest struct {
}

type OncallSetTest struct {
}

func (o *OncallGetTest) getAlertgroups(groups *[][]byte, state string, page string, pagenum int) error {
	return nil
}

func (o *OncallGetTest) getActiveAlertgroups() ([][]byte, error) {
	return [][]byte{
		[]byte(`[
		{"id":"1234","title":"Test Alert","alert_ids":["1","2","3"]}
		]`),
	}, nil
}

func (o *OncallGetTest) getAlertgroupAlertsByGroupID(alertGroupID string) ([][]byte, error) {
	return [][]byte{
		[]byte(`
		[{"id":"1","payload": "{\"alertmanager_origin_alerts\":[]}"}]
		`),
		[]byte(`
		[{"id":"2","payload": "{\"alertmanager_origin_alerts\":[]}"}]
		`),
		[]byte(`
		[{"id":"3","payload": "{\"alertmanager_origin_alerts\":[{\"fingerprint"\:\"1\",\"labels\":{\"alertname\":\"Test Alert\"},\"annotations\":{}}]}"}]
		`),
	}, nil
}

func (o *OncallSetTest) doOncallIncident(oncall OncallRequest) error {
	return nil
}

func TestSendAlerts(t *testing.T) {
	o := OncallSink{runbooks: &runbooks, oncallGet: &OncallGetTest{}, oncallSet: &OncallSetTest{}}
	o.runbooks.OncallMessage.Title = "{{.Labels.alertname}}"

	alerts := []sharedtools.Alert{
		{
			Title:       "Test Alert",
			Fingerprint: "1",
			Labels: map[string]string{
				"alertname": "Test Alert",
				"severity":  "info",
			},
			Annotations: map[string]string{},
		},
		{
			Title:       "Test Alert",
			Fingerprint: "2",
			Labels: map[string]string{
				"alertname": "Test Alert",
				"severity":  "warning",
			},
			Annotations: map[string]string{},
		},
		{
			Title:       "Test Alert",
			Fingerprint: "3",
			Labels: map[string]string{
				"alertname": "Test Alert",
				"severity":  "warning",
			},
			Annotations: map[string]string{},
		},
	}
	accepted, resolved, errors := o.SendAlerts(alerts)

	if len(accepted) != 3 {
		t.Errorf("expected accepted length to be 3, got %d", len(accepted))
	}
	if len(resolved) != 0 {
		t.Errorf("expected resolved length to be 1, got %d", len(resolved))
	}
	if len(errors) > 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}
}

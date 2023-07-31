package alertsource

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/mobalyticshq/alertsforge/config"
	"github.com/mobalyticshq/alertsforge/sharedtools"
	"github.com/stretchr/testify/assert"
)

type mockSink struct {
	ReceivedAlerts sharedtools.AlertsSlice
}

func (m *mockSink) SendAlerts(alerts []sharedtools.Alert) (accepted []string, resolved []string, errors []error) {
	m.ReceivedAlerts = alerts
	return []string{"alert1", "alert2", "alert5", "alert6"}, []string{"alert3"}, []error{http.ErrContentLength}
}

type mockEnricher struct {
}

func (e *mockEnricher) StartEnrichmentFlow(alert sharedtools.Alert) []error {
	if alert.Fingerprint == "alert5" {
		return []error{fmt.Errorf("alert5 got error on enriching")}
	}
	return nil
}

func TestAlertManager_ProcessAlertsBuffer(t *testing.T) {

	// create an AlertManager instance
	am := &AlertManager{
		AlertsBuffer: map[string]*sharedtools.Alert{
			"alert1": {
				Fingerprint: "alert1",
				StartsAt:    time.Now().Add(-1 * time.Hour),
				EndsAt:      time.Now().Add(1 * time.Hour),
				Status:      sharedtools.Pending,
			},
			"alert2": {
				Fingerprint: "alert2",
				StartsAt:    time.Now().Add(-2 * time.Hour),
				EndsAt:      time.Now().Add(1 * time.Hour),
				Status:      sharedtools.Pending,
			},
			"alert3": {
				Fingerprint: "alert3",
				StartsAt:    time.Now().Add(-3 * time.Hour),
				EndsAt:      time.Now().Add(-1 * time.Hour),
				Status:      sharedtools.Firing,
			},
			"alert4": {
				Fingerprint: "alert4",
				StartsAt:    time.Now().Add(-4 * time.Hour),
				EndsAt:      time.Now().Add(-1 * time.Hour),
				Status:      sharedtools.Pending,
			},
			"alert5": {
				Fingerprint: "alert5",
				StartsAt:    time.Now().Add(-5 * time.Hour),
				EndsAt:      time.Now().Add(5 * time.Hour),
				Status:      sharedtools.Pending,
			},
			"alert6": {
				Fingerprint: "alert6",
				StartsAt:    time.Now().Add(-6 * time.Hour),
				EndsAt:      time.Now().Add(6 * time.Hour),
				LastSinkAt:  time.Now().Add(-2 * time.Hour),
				Status:      sharedtools.Firing,
			},
		},
		AlertBufferMutex: sync.RWMutex{},
		AlertSink:        &mockSink{},
		AlertEnricher:    &mockEnricher{},
		runbooks:         &config.RunbooksConfig{},
	}

	os.Setenv("AF_RESINK_TIME", "30m")
	errs := am.ProcessAlertsBuffer()

	assert.ElementsMatch(t, []error{http.ErrContentLength}, errs)

	alertsSentToSink := am.AlertSink.(*mockSink).ReceivedAlerts
	sort.Sort(alertsSentToSink)
	assert.Equal(t, 5, len(alertsSentToSink))
	assert.Equal(t, "alert1", alertsSentToSink[0].Fingerprint)
	assert.Equal(t, "alert2", alertsSentToSink[1].Fingerprint)
	assert.Equal(t, "alert3", alertsSentToSink[2].Fingerprint)
	assert.Equal(t, "alert5", alertsSentToSink[3].Fingerprint)
	assert.Equal(t, "alert6", alertsSentToSink[4].Fingerprint)

	// assert that alert 3 was not sent to the alert sink
	_, ok := am.AlertsBuffer["alert3"]
	assert.True(t, !ok, "Alert 3 was removed from buffer")
}

func TestAsJSON(t *testing.T) {
	testCases := []struct {
		name          string
		status        int
		message       string
		expectedJSON  string
		expectedError bool
	}{
		{
			name:          "Success Response",
			status:        http.StatusOK,
			message:       "success",
			expectedJSON:  `{"Status":200,"Message":"success"}`,
			expectedError: false,
		},
		{
			name:          "Error Response",
			status:        http.StatusInternalServerError,
			message:       "internal server error",
			expectedJSON:  `{"Status":500,"Message":"internal server error"}`,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock response writer
			writer := httptest.NewRecorder()

			// Call the asJson function
			asJson(writer, tc.status, tc.message)

			// Check if the returned JSON is correct
			jsonResponse := writer.Body.String()
			if tc.expectedJSON != jsonResponse {
				t.Errorf("Expected JSON %v, but got %v", tc.expectedJSON, jsonResponse)
			}

			// Check if the appropriate status code was set
			if writer.Result().StatusCode != tc.status {
				t.Errorf("Expected status code %v, but got %v", tc.status, writer.Code)
			}

			// Check for errors
			if tc.expectedError && writer.Body.Len() == 0 {
				t.Error("Expected an error, but the response was empty")
			} else if !tc.expectedError && writer.Body.Len() == 0 {
				t.Error("Expected a valid response, but got an error")
			}
		})
	}
}

func TestAlertManager_receiveAlerts(t *testing.T) {

	t.Run("silences alert", func(t *testing.T) {
		am := &AlertManager{
			runbooks: &config.RunbooksConfig{
				Silences: []config.Silence{
					{
						LabelsSelector: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			AlertsBuffer:     map[string]*sharedtools.Alert{},
			AlertBufferMutex: sync.RWMutex{},
		}
		am.receiveAlerts([]sharedtools.Alert{
			{
				Labels: map[string]string{
					"foo": "bar",
				},
			},
		})

		if len(am.AlertsBuffer) > 0 {
			t.Errorf("expected no alerts in buffer but got: %v", am.AlertsBuffer)
		}
	})

	t.Run("delayed resolve with label", func(t *testing.T) {
		am := &AlertManager{
			runbooks:         &config.RunbooksConfig{},
			AlertsBuffer:     map[string]*sharedtools.Alert{},
			AlertBufferMutex: sync.RWMutex{},
		}
		alertsforge_delay_resolve := "5m"
		alertTime := time.Now()

		am.receiveAlerts([]sharedtools.Alert{
			{
				Labels: map[string]string{
					"alertsforge_delay_resolve": alertsforge_delay_resolve,
				},
				EndsAt: alertTime,
			},
		})

		if len(am.AlertsBuffer) == 0 {
			t.Errorf("expected 1 alert in buffer but got: %v", am.AlertsBuffer)
		} else {
			alert := am.AlertsBuffer["61a9361fbe6f8976"]
			add, _ := time.ParseDuration(alertsforge_delay_resolve)
			assert.Equal(t, alertTime.Add(add), alert.EndsAt)
		}
	})

	t.Run("delayed resolve with env", func(t *testing.T) {
		am := &AlertManager{
			runbooks:         &config.RunbooksConfig{},
			AlertsBuffer:     map[string]*sharedtools.Alert{},
			AlertBufferMutex: sync.RWMutex{},
		}
		alertsforge_delay_resolve := "10m"
		os.Setenv("AF_DEFAULT_RESOLVE_DELAY", alertsforge_delay_resolve)
		alertTime := time.Now()

		am.receiveAlerts([]sharedtools.Alert{
			{
				Labels: map[string]string{
					"somelabel": "value",
				},
				EndsAt: alertTime,
			},
		})

		if len(am.AlertsBuffer) == 0 {
			t.Errorf("expected 1 alert in buffer but got: %v", am.AlertsBuffer)
		} else {
			alert := am.AlertsBuffer["c337993c31eb8eac"]
			add, _ := time.ParseDuration(alertsforge_delay_resolve)
			assert.Equal(t, alertTime.Add(add), alert.EndsAt)
		}
	})

	t.Run("delayed resolve incorrect", func(t *testing.T) {
		am := &AlertManager{
			runbooks:         &config.RunbooksConfig{},
			AlertsBuffer:     map[string]*sharedtools.Alert{},
			AlertBufferMutex: sync.RWMutex{},
		}
		alertsforge_delay_resolve := "5mm"
		os.Setenv("AF_DEFAULT_RESOLVE_DELAY", alertsforge_delay_resolve)
		alertTime := time.Now()

		am.receiveAlerts([]sharedtools.Alert{
			{
				Labels: map[string]string{
					"somelabel": "value",
				},
				EndsAt: alertTime,
			},
		})

		if len(am.AlertsBuffer) == 0 {
			t.Errorf("expected 1 alert in buffer but got: %v", am.AlertsBuffer)
		} else {
			alert := am.AlertsBuffer["c337993c31eb8eac"]
			assert.Equal(t, alertTime, alert.EndsAt)
		}
	})

	t.Run("update existing alert", func(t *testing.T) {
		alertTime := time.Now()
		am := &AlertManager{
			runbooks: &config.RunbooksConfig{},
			AlertsBuffer: map[string]*sharedtools.Alert{"c337993c31eb8eac": {
				Labels: map[string]string{
					"somelabel": "value",
				},
				EndsAt: alertTime,
			}},
			AlertBufferMutex: sync.RWMutex{},
		}

		add, _ := time.ParseDuration("20m")
		am.receiveAlerts([]sharedtools.Alert{
			{
				Labels: map[string]string{
					"somelabel": "value",
				},
				EndsAt: alertTime.Add(add),
			},
		})

		if len(am.AlertsBuffer) == 0 {
			t.Errorf("expected 1 alert in buffer but got: %v", am.AlertsBuffer)
		} else {
			alert := am.AlertsBuffer["c337993c31eb8eac"]
			assert.Equal(t, alertTime.Add(add), alert.EndsAt)
		}
	})
}

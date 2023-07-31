package enrichers

import (
	"errors"
	"reflect"
	"testing"
)

func TestStaticEnricher(t *testing.T) {
	alertinfo := AlertInfo{
		Labels: map[string]string{
			"alertname": "TestAlert",
			"instance":  "server1",
		},
		Annotations: map[string]string{
			"summary":     "Test alert summary",
			"description": "Test alert description",
		},
	}
	enricher := &staticEnricher{alertinfo: alertinfo}

	tests := []struct {
		name          string
		config        map[string]string
		expected      map[string]string
		expectedError error
	}{
		{
			name: "valid config",
			config: map[string]string{
				"targetLabel": "enriched_summary",
				"value":       "summary",
			},
			expected: map[string]string{
				"enriched_summary": "summary",
			},
			expectedError: nil,
		},
		{
			name: "missing targetLabel",
			config: map[string]string{
				"value": "summary",
			},
			expected:      nil,
			expectedError: errors.New("not enough config parameters, 'targetLabel' is mandatory"),
		},
		{
			name: "missing value",
			config: map[string]string{
				"targetLabel": "enriched_summary",
			},
			expected:      nil,
			expectedError: errors.New("not enough config parameters, 'value' is mandatory"),
		},
		{
			name: "empty value",
			config: map[string]string{
				"targetLabel": "enriched_summary",
				"value":       "",
			},
			expected:      nil,
			expectedError: nil,
		},
		{
			name: "invalid value template",
			config: map[string]string{
				"targetLabel": "enriched_summary",
				"value":       "{{invalid_template}",
			},
			expected:      nil,
			expectedError: errors.New("template: value:1: bad character U+007D '}'"),
		},
		{
			name: "correct template",
			config: map[string]string{
				"targetLabel": "enriched_summary",
				"value":       "{{.Annotations.description}}",
			},
			expected:      map[string]string{"enriched_summary": "Test alert description"},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enricher.config = test.config
			result, err := enricher.Enrich()
			if err != nil && err.Error() != test.expectedError.Error() {
				t.Errorf("Expected error '%s' but got '%s'", test.expectedError, err)
			}
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Expected '%v' but got '%v'", test.expected, result)
			}
		})
	}
}

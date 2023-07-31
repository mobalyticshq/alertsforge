package enrichers

import (
	"fmt"
	"reflect"
	"testing"
)

type fileReaderTest struct{}

func (c *fileReaderTest) getFile(filePath string) ([]byte, error) {
	return []byte(`environment: 'test environment value'`), nil
}

type fileReaderErrTest struct{}

func (c *fileReaderErrTest) getFile(filePath string) ([]byte, error) {
	return nil, fmt.Errorf("file not found")
}

type fileReaderYamlErrTest struct{}

func (c *fileReaderYamlErrTest) getFile(filePath string) ([]byte, error) {
	return []byte(`environment= 'test environment value'`), nil
}
func TestEnrich(t *testing.T) {
	// Mock configuration to use in tests
	config := map[string]string{
		fileName:    "testdata/variables.yaml",
		targetLabel: "enriched_label",
		value:       "{{ .Variables.environment }}",
	}
	// Mock alertinfo to use in tests
	alertinfo := AlertInfo{
		Labels: map[string]string{
			"alertname": "TestAlert",
			"instance":  "testing",
			"severity":  "critical",
		},
		Annotations: map[string]string{
			"message": "This is a test alert",
		},
		StartsAt: "2021-01-01T00:00:00.000Z",
	}

	tests := []struct {
		name         string
		yamlEnricher *yamlEnricher
		want         map[string]string
		wantErr      bool
	}{
		{
			name:         "should return enriched label value",
			yamlEnricher: &yamlEnricher{config: config, alertinfo: alertinfo, fileReader: &fileReaderTest{}},
			want:         map[string]string{"enriched_label": "test environment value"},
		},
		{
			name: "should return empty map if value not exist",
			yamlEnricher: &yamlEnricher{config: map[string]string{
				fileName:    "testdata/variables.yaml",
				targetLabel: "enriched_label",
				value:       "{{ .Variables.undefined }}",
			},
				alertinfo:  alertinfo,
				fileReader: &fileReaderTest{},
			},
			want: nil,
		},
		{
			name: "should return error if target label config parameter is missing",
			yamlEnricher: &yamlEnricher{config: map[string]string{
				fileName: "testdata/variables.yaml",
				value:    "{{ .Variables.environment }}",
			},
				alertinfo:  alertinfo,
				fileReader: &fileReaderTest{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return error if file not found",
			yamlEnricher: &yamlEnricher{config: map[string]string{
				fileName:    "undefined.yaml",
				targetLabel: "enriched_label",
				value:       "{{ .Variables.environment }}",
			},
				alertinfo:  alertinfo,
				fileReader: &fileReaderErrTest{},
			},

			want:    nil,
			wantErr: true,
		},
		{
			name: "should return error if YAML unmarshaling fails",
			yamlEnricher: &yamlEnricher{config: map[string]string{
				fileName:    "testdata/invalid_yaml.yaml",
				targetLabel: "enriched_label",
				value:       "{{ .Variables.environment }}",
			},
				alertinfo:  alertinfo,
				fileReader: &fileReaderYamlErrTest{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return error if template string parsing fails",
			yamlEnricher: &yamlEnricher{config: map[string]string{
				fileName:    "testdata/variables.yaml",
				targetLabel: "enriched_label",
				value:       "{{{ .Variables.undefined }}",
			},
				alertinfo:  alertinfo,
				fileReader: &fileReaderTest{}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.yamlEnricher.Enrich()
			if (err != nil) != tt.wantErr {
				t.Errorf("yamlEnricher.Enrich() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("yamlEnricher.Enrich() = %v, want %v", got, tt.want)
			}
		})
	}
}

package sharedtools

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/dlclark/regexp2"
	"go.uber.org/zap"
)

const (
	Pending  = "pending"
	Firing   = "firing"
	Resolved = "resolved"

	offset64           = 14695981039346656037
	prime64            = 1099511628211
	SeparatorByte byte = 255
)

type Alert struct {
	Status        string            `json:"status"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	StartsAt      time.Time         `json:"startsAt"`
	EndsAt        time.Time         `json:"endsAt"`
	GeneratorURL  string            `json:"generatorURL"`
	Fingerprint   string            `json:"fingerprint"`
	Title         string            `json:"title"`
	LastSinkAt    time.Time         `json:"-"`
	LastReceiveAt time.Time         `json:"-"`
}

type AlertsSlice []Alert

func (a AlertsSlice) Len() int           { return len(a) }
func (a AlertsSlice) Less(i, j int) bool { return a[i].StartsAt.Unix() > a[j].StartsAt.Unix() }
func (a AlertsSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func MustTemplateString(tpl string, variables any, onerror string) string {
	parsedValue, err := TemplateString(tpl, variables)
	if err != nil {
		return onerror
	}
	return parsedValue
}

func TemplateString(tpl string, variables any) (string, error) {
	parsedtemplate, err := template.New("value").Funcs(sprig.FuncMap()).Option("missingkey=error").Parse(tpl)
	if err != nil {
		zap.S().Errorf("template error:", err)
		return "", err
	}

	parsedValue := new(bytes.Buffer)
	if err := parsedtemplate.Execute(parsedValue, variables); err != nil && !strings.Contains(err.Error(), "map has no entry for key") {
		zap.S().Errorf("template error:", err)
		return "", err
	}
	/*if len(parsedValue.String()) == 0 {
		return "", errors.New("zero string while parsing")
	}*/
	return parsedValue.String(), nil
}

func LabelSetToFingerprint(labels map[string]string) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%016x", uint64(offset64))
	}

	labelNames := make([]string, 0, len(labels))
	for labelName := range labels {
		labelNames = append(labelNames, labelName)
	}
	sort.Strings(labelNames)

	sum := hashNew()
	for _, labelName := range labelNames {
		sum = hashAdd(sum, labelName)
		sum = hashAddByte(sum, SeparatorByte)
		sum = hashAdd(sum, labels[labelName])
		sum = hashAddByte(sum, SeparatorByte)
	}

	return fmt.Sprintf("%016x", sum)
}

// hashNew initializes a new fnv64a hash value.
func hashNew() uint64 {
	return offset64
}

// hashAdd adds a string to a fnv64a hash value, returning the updated hash.
func hashAdd(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}

// hashAddByte adds a byte to a fnv64a hash value, returning the updated hash.
func hashAddByte(h uint64, b byte) uint64 {
	h ^= uint64(b)
	h *= prime64
	return h
}

func CopyMap(m map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func CopyAlert(alert *Alert) Alert {
	return Alert{
		Status:        alert.Status,
		Labels:        CopyMap(alert.Labels),
		Annotations:   CopyMap(alert.Annotations),
		StartsAt:      alert.StartsAt,
		EndsAt:        alert.EndsAt,
		GeneratorURL:  alert.GeneratorURL,
		Fingerprint:   alert.Fingerprint,
		LastSinkAt:    alert.LastSinkAt,
		LastReceiveAt: alert.LastReceiveAt,
	}
}

func MatchLabels(alertLabels, labelsSelector map[string]string) bool {
	notMatchedLabels := make(map[string]string, 0)

	for selectorKey, selectorValue := range labelsSelector {
		if alertLabelValue, ok := alertLabels[selectorKey]; ok {
			if selectorValue == "" && alertLabelValue == "" {
				continue
			}
			if selectorValue != "" {
				regExp := regexp2.MustCompile(selectorValue, 0)
				if isMatch, _ := regExp.MatchString(alertLabelValue); isMatch {
					continue
				}
			}
		} else if selectorValue == "" {
			continue
		}
		notMatchedLabels[selectorKey] = selectorValue
	}
	return len(notMatchedLabels) == 0
}

func MergeMaps(oldMap, newMap map[string]string) {
	for key, value := range newMap {
		oldMap[key] = value
	}
}

type HTTPInterface interface {
	FetchResponse(req *http.Request) ([]byte, error)
}

type HTTPClient struct{}

func (c *HTTPClient) FetchResponse(req *http.Request) ([]byte, error) {

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return resBody, nil
}

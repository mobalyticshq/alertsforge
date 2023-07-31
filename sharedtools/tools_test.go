package sharedtools

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestMergeMaps(t *testing.T) {
	t.Run("merging two empty maps yields an empty map", func(t *testing.T) {
		oldMap := map[string]string{}
		newMap := map[string]string{}
		expectedMap := map[string]string{}

		MergeMaps(oldMap, newMap)

		if !reflect.DeepEqual(oldMap, expectedMap) {
			t.Errorf("expected %v, but got %v", expectedMap, oldMap)
		}
	})

	t.Run("merging a non-empty map into an empty map yields the non-empty map", func(t *testing.T) {
		oldMap := map[string]string{}
		newMap := map[string]string{"foo": "bar", "baz": "qux"}
		expectedMap := map[string]string{"foo": "bar", "baz": "qux"}

		MergeMaps(oldMap, newMap)

		if !reflect.DeepEqual(oldMap, expectedMap) {
			t.Errorf("expected %v, but got %v", expectedMap, oldMap)
		}
	})

	t.Run("merging a non-empty map into a non-empty map overwrites existing values", func(t *testing.T) {
		oldMap := map[string]string{"foo": "old value"}
		newMap := map[string]string{"foo": "new value", "baz": "qux"}
		expectedMap := map[string]string{"foo": "new value", "baz": "qux"}

		MergeMaps(oldMap, newMap)

		if !reflect.DeepEqual(oldMap, expectedMap) {
			t.Errorf("expected %v, but got %v", expectedMap, oldMap)
		}
	})
}

func TestMatchLabels(t *testing.T) {

	t.Run("Test case where all labels match", func(t *testing.T) {
		alertLabels := map[string]string{
			"app":     "myapp",
			"version": "1.0",
			"env":     "prod",
		}
		labelsSelector := map[string]string{
			"version": "1.0",
			"env":     "prod",
			"app":     "myapp",
		}
		if !MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected true but got false")
		}
	})
	t.Run("Test case where some labels don't match", func(t *testing.T) {
		alertLabels := map[string]string{
			"app":     "myapp",
			"version": "1.0",
			"env":     "prod",
		}
		labelsSelector := map[string]string{
			"version": "2.0",
			"env":     "prod",
			"app":     "myapp",
		}
		if MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected false but got true")
		}
	})

	t.Run("Test case where alertLabels param is empty", func(t *testing.T) {
		alertLabels := map[string]string{}
		labelsSelector := map[string]string{
			"version": "1.0",
		}
		if MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected false but got true")
		}
	})

	t.Run("Test case where  labelsSelector param is empty", func(t *testing.T) {
		alertLabels := map[string]string{
			"version": "1.0",
		}
		labelsSelector := map[string]string{}
		if !MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected true but got false")
		}
	})

	t.Run("Test case where some selector labels empty and it does not exists in labels", func(t *testing.T) {
		alertLabels := map[string]string{
			"version": "1.0",
			"env":     "prod",
		}
		labelsSelector := map[string]string{
			"app":     "",
			"version": "1.0",
			"env":     "prod",
		}
		if !MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected true but got false")
		}
	})

	t.Run("Test case where some selector labels empty and it does exists in labels", func(t *testing.T) {
		alertLabels := map[string]string{
			"app":     "some",
			"version": "1.0",
			"env":     "prod",
		}
		labelsSelector := map[string]string{
			"app":     "",
			"version": "2.0",
			"env":     "prod",
		}

		if MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected false but got true")
		}
	})

	t.Run("Test case where some selector labels empty and label value empty", func(t *testing.T) {
		alertLabels := map[string]string{
			"app":     "",
			"version": "1.0",
			"env":     "prod",
		}
		labelsSelector := map[string]string{
			"app":     "",
			"version": "1.0",
			"env":     "prod",
		}
		if !MatchLabels(alertLabels, labelsSelector) {
			t.Errorf("MatchLabels failed, expected true but got false")
		}
	})
}

func TestLabelSetToFingerprint(t *testing.T) {

	t.Run("Test empty label set", func(t *testing.T) {
		labels := make(map[string]string)
		expected := "cbf29ce484222325"
		result := LabelSetToFingerprint(labels)
		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("Test one label", func(t *testing.T) {
		labels := map[string]string{"foo": "bar"}
		expected := "3fff2c2d7595e046"
		result := LabelSetToFingerprint(labels)
		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("Test multiple labels", func(t *testing.T) {
		labels := map[string]string{"foo": "bar", "baz": "qux"}
		expected := "e4d83091cd448e77"
		result := LabelSetToFingerprint(labels)
		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("Test labels with different order", func(t *testing.T) {
		labels1 := map[string]string{"foo": "bar", "baz": "qux"}
		labels2 := map[string]string{"baz": "qux", "foo": "bar"}
		result1 := LabelSetToFingerprint(labels1)
		result2 := LabelSetToFingerprint(labels2)
		if result1 != result2 {
			t.Errorf("Expected %s, but got %s", result1, result2)
		}
	})

}

func TestCopyMap(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		m1 := make(map[string]string)
		m2 := CopyMap(m1)
		if !reflect.DeepEqual(m1, m2) {
			t.Errorf("CopyMap(%v) = %v, want %v", m1, m2, m1)
		}
	})

	t.Run("non-empty map", func(t *testing.T) {
		m1 := map[string]string{"foo": "bar", "baz": "qux"}
		m2 := CopyMap(m1)
		if !reflect.DeepEqual(m1, m2) {
			t.Errorf("CopyMap(%v) = %v, want %v", m1, m2, m1)
		}
	})

	t.Run("map with nil values", func(t *testing.T) {
		m1 := map[string]string{"foo": "bar", "baz": ""}
		m2 := CopyMap(m1)
		if !reflect.DeepEqual(m1, m2) {
			t.Errorf("CopyMap(%v) = %v, want %v", m1, m2, m1)
		}
	})

	t.Run("source map changed values", func(t *testing.T) {
		m1 := map[string]string{"foo": "bar", "baz": "baz"}
		m2 := CopyMap(m1)
		m1["foo"] = "bar2"
		if reflect.DeepEqual(m1, m2) {
			t.Errorf("Maps should not reflect changes but they are equal %v, %v", m2, m1)
		}
	})

}

func TestMustTemplateString(t *testing.T) {
	variables := struct{ Name string }{Name: "John"}
	t.Run("Test case for a valid template string and variables", func(t *testing.T) {

		tpl := "Hello {{.Name}}, how are you doing?"
		onerror := ""

		result := MustTemplateString(tpl, variables, onerror)

		if result != "Hello John, how are you doing?" {
			t.Errorf("Expected 'Hello John, how are you doing?', got %s", result)
		}
	})
	t.Run("Test case for an invalid template string and a default value on error", func(t *testing.T) {
		tpl := "Hello {{.NonexistentField}}, how are you doing?"
		onerror := "Uh oh, something went wrong."
		result := MustTemplateString(tpl, variables, onerror)
		if result != onerror {
			t.Errorf("Expected '%s', got %s", onerror, result)
		}
	})

	t.Run("Test case for an invalid template and a default value on error", func(t *testing.T) {
		tpl := "Hello {{.errorous template}, how are you doing?"
		onerror := "Uh oh, something went wrong."
		result := MustTemplateString(tpl, variables, onerror)
		if result != onerror {
			t.Errorf("Expected '%s', got %s", onerror, result)
		}
	})
}

func TestCopyAlert(t *testing.T) {
	alert := &Alert{
		Status:       "firing",
		Labels:       map[string]string{"severity": "critical", "service": "web"},
		Annotations:  map[string]string{"summary": "High CPU utilization"},
		StartsAt:     time.Now(),
		EndsAt:       time.Now().Add(5 * time.Minute),
		GeneratorURL: "https://example.com/alerts",
		Fingerprint:  "12345",
	}

	// Test if the copied alert is equal to the original
	copiedAlert := CopyAlert(alert)
	if !reflect.DeepEqual(alert, &copiedAlert) {
		t.Error("Copied alert is not equal to the original alert")
	}

	// Check if the copied alert's labels and annotations are different from the original
	alert.Labels["severity"] = "low"
	alert.Annotations["summary"] = "Low CPU utilization"
	if reflect.DeepEqual(alert.Labels, copiedAlert.Labels) || reflect.DeepEqual(alert.Annotations, copiedAlert.Annotations) {
		t.Error("Copied alert is identical to the original in labels or annotations")
	}
}

func TestAlertsSlice(t *testing.T) {
	alert1 := Alert{StartsAt: time.Unix(1626385927, 0)}
	alert2 := Alert{StartsAt: time.Unix(1626386928, 0)}
	alert3 := Alert{StartsAt: time.Unix(1626387929, 0)}

	// test cases
	t.Run("Less() should return true for alert1 and alert2", func(t *testing.T) {
		alertsSlice := AlertsSlice{alert1, alert2}
		got := alertsSlice.Less(0, 1)
		if got {
			t.Errorf("alertsSlice.Less(0, 1) = %v; expected true", got)
		}
	})

	t.Run("Less() should return false for alert2 and alert1", func(t *testing.T) {
		alertsSlice := AlertsSlice{alert1, alert2}
		got := alertsSlice.Less(1, 0)
		if !got {
			t.Errorf("alertsSlice.Less(1, 0) = %v; expected false", got)
		}
	})

	t.Run("Sort() should order alerts by start time in descending order", func(t *testing.T) {
		alertsSlice := AlertsSlice{alert2, alert1, alert3}
		sort.Sort(alertsSlice)
		got := []Alert{alertsSlice[0], alertsSlice[1], alertsSlice[2]}
		expected := []Alert{alert3, alert2, alert1}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("alertSlice.Sort() = %v; expected %v", got, expected)
		}
	})
}

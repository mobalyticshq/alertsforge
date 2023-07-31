package enrichers

import (
	"testing"
	"time"

	"github.com/mobalyticshq/alertsforge/sharedtools"
	"github.com/stretchr/testify/assert"
)

func TestCommandEnricher_Enrich_Success(t *testing.T) {
	c := commandEnricher{
		config: map[string]string{
			command:            "echo 'hello {{ .Labels.label1}}'",
			targetLabelsPrefix: "test_prefix",
		},
		alertinfo: AlertInfo{Labels: map[string]string{"label1": "world"}},
	}

	result, err := c.Enrich()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, map[string]string{
		"test_prefix_stdout": "hello world\n",
	}, result)
}

func TestCommandEnricher_Enrich_Failure(t *testing.T) {
	c := commandEnricher{
		config: map[string]string{
			command:            "echoo 'hello {{ .Labels.label1}}'",
			targetLabelsPrefix: "test_prefix",
		},
		alertinfo: AlertInfo{Labels: map[string]string{"label1": "world"}},
	}

	result, err := c.Enrich()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result["test_prefix_stderr"], "echoo: not found")
}

func TestCommandEnricher_Parameters_Failure(t *testing.T) {
	c := commandEnricher{
		config: map[string]string{
			command:            "echo 'hello {{{ .Labels.label1}}'",
			targetLabelsPrefix: "test_prefix",
		},
		alertinfo: AlertInfo{Labels: map[string]string{"label1": "world"}},
	}

	_, err := c.Enrich()

	assert.Error(t, err)

}

func TestCommandEnricher_Template_Failure(t *testing.T) {
	c := commandEnricher{
		config: map[string]string{
			targetLabelsPrefix: "test_prefix",
		},
		alertinfo: AlertInfo{Labels: map[string]string{"label1": "world"}},
	}

	_, err := c.Enrich()

	assert.Error(t, err)

}

type bucketWriterTest struct {
	result string
}

func (b *bucketWriterTest) writeToBucket(bucketName, stdOutFilename string, data []byte) error {
	b.result = string(data)
	return nil
}

func TestCommandEnricher_EnrichToBucket_Success(t *testing.T) {
	bw := &bucketWriterTest{result: "somedata"}
	c := &commandEnricher{
		config: map[string]string{
			command:            "echo 'hello {{ .Labels.label1}}'",
			targetLabelsPrefix: "test_prefix",
			bucket:             "testbucket",
		},
		alertinfo:    AlertInfo{Labels: map[string]string{"label1": "world"}},
		bucketWriter: bw,
	}

	result, err := c.Enrich()

	filename := time.Now().Format("2006-01-02") + "/" + sharedtools.LabelSetToFingerprint(c.alertinfo.Labels) + sharedtools.LabelSetToFingerprint(c.config) + "_stdout.txt"
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, map[string]string{
		"test_prefix_stdout": filename,
	}, result)
	assert.Equal(t, "hello world\n", bw.result)
}

func TestCommandEnricher_EnrichToBucket_Fail(t *testing.T) {
	bw := &bucketWriterTest{result: "somedata"}
	c := &commandEnricher{
		config: map[string]string{
			command:            "echoo 'hello {{ .Labels.label1}}'",
			targetLabelsPrefix: "test_prefix",
			bucket:             "testbucket",
		},
		alertinfo:    AlertInfo{Labels: map[string]string{"label1": "world"}},
		bucketWriter: bw,
	}

	result, err := c.Enrich()

	filename := time.Now().Format("2006-01-02") + "/" + sharedtools.LabelSetToFingerprint(c.alertinfo.Labels) + sharedtools.LabelSetToFingerprint(c.config) + "_stderr.txt"
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, map[string]string{
		"test_prefix_stderr": filename,
	}, result)
	assert.Contains(t, bw.result, "echoo: not found")
}

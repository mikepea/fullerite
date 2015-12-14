package handler

import (
	"fullerite/metric"
	l "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func getTestDebugHandler(interval int, buffsize int) *Debug {
	testChannel := make(chan metric.Metric)
	testLog := l.WithField("testing", "debug_handler")

	return NewDebug(testChannel, interval, buffsize, testLog)
}

func TestDebugConfigureEmptyConfig(t *testing.T) {
	config := make(map[string]interface{})
	h := getTestDebugHandler(12, 13)
	h.Configure(config)

	assert.Equal(t, 12, h.Interval())
	assert.Equal(t, 13, h.MaxBufferSize())
}

func TestDebugConfigure(t *testing.T) {
	config := map[string]interface{}{
		"interval":        "10",
		"max_buffer_size": "100",
	}

	h := getTestDebugHandler(12, 13)
	h.Configure(config)

	assert.Equal(t, 10, h.Interval())
	assert.Equal(t, 100, h.MaxBufferSize())
}

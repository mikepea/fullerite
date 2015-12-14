package handler

import (
	"encoding/json"
	"fmt"
	"fullerite/metric"

	l "github.com/Sirupsen/logrus"
)

// Debug type
type Debug struct {
	BaseHandler
}

// NewDebug returns a new Debug handler.
func NewDebug(
	channel chan metric.Metric,
	initialInterval int,
	initialBufferSize int,
	log *l.Entry) *Debug {

	inst := new(Debug)
	inst.name = "Debug"

	inst.interval = initialInterval
	inst.maxBufferSize = initialBufferSize
	inst.log = log
	inst.channel = channel

	return inst
}

// Configure accepts the different configuration options for the Debug handler
func (h *Debug) Configure(configMap map[string]interface{}) {
	h.configureCommonParams(configMap)
}

// Run runs the handler main loop
func (h *Debug) Run() {
	h.run(h.emitMetrics)
}

func (h *Debug) convertToDebug(incomingMetric metric.Metric) string {
	json_out, _ := json.Marshal(incomingMetric)
	return string(json_out)
}

func (h *Debug) emitMetrics(metrics []metric.Metric) bool {
	h.log.Info("Starting to emit ", len(metrics), " metrics")

	if len(metrics) == 0 {
		h.log.Warn("Skipping send because of an empty payload")
		return false
	}

	for _, m := range metrics {
		h.log.Info(fmt.Sprintf(h.convertToDebug(m)))
	}
	return true
}

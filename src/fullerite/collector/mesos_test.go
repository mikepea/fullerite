package collector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"fullerite/metric"

	l "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// mockExternalIP Injectable mock for externalIP, for test assertions.
func mockExternalIP() (string, error) {
	return httptest.DefaultRemoteAddr, nil
}

func TestMesosStatsNewMesosStats(t *testing.T) {
	oldExternalIP := externalIP
	defer func() { externalIP = oldExternalIP }()

	externalIP = mockExternalIP

	c := make(chan metric.Metric)
	i := 10
	l := defaultLog.WithFields(l.Fields{"collector": "Mesos"})

	sut := newMesosStats(c, i, l).(*MesosStats)

	assert.Equal(t, c, sut.channel)
	assert.Equal(t, i, sut.interval)
	assert.Equal(t, l, sut.log)
	assert.Equal(t, httptest.DefaultRemoteAddr, sut.IP)
	assert.Equal(t, http.Client{Timeout: getTimeout}, sut.client)
}

func TestMesosStatsCollect(t *testing.T) {
	oldExternalIP := externalIP
	oldSendMetrics := sendMetrics
	defer func() {
		externalIP = oldExternalIP
		sendMetrics = oldSendMetrics
	}()

	sendMetricsCalled := false
	c := make(chan bool)
	sendMetrics = func(m *MesosStats) {
		sendMetricsCalled = true
		c <- true
	}

	tests := []struct {
		configMap           map[string]interface{}
		externalIP          string
		isSendMetricsCalled bool
		msg                 string
	}{
		{map[string]interface{}{"mesosNodes": "ip1,ip2"}, "5.6.7.8", true, "Machine IP is not equal to leader IP, therefore we should skip collection."},
		{map[string]interface{}{"mesosNodes": "ip1,ip2"}, httptest.DefaultRemoteAddr, true, "Current box is leader; therefore, we should be called sendMetrics."},
	}

	for _, test := range tests {
		sendMetricsCalled = false
		configMap := test.configMap
		externalIP = func() (string, error) { return test.externalIP, nil }

		sut := newMesosStats(nil, 0, defaultLog).(*MesosStats)
		sut.Configure(configMap)
		sut.Collect()

		switch test.isSendMetricsCalled {
		case false:
			assert.False(t, sendMetricsCalled, test.msg)
		case true:
			<-c
			assert.True(t, sendMetricsCalled, test.msg)
		}
	}
}

func TestMesosStatsSendMetrics(t *testing.T) {
	oldGetMetrics := getMetrics
	defer func() { getMetrics = oldGetMetrics }()

	expected := metric.Metric{"mesos.test", "gauge", 0.1, map[string]string{}}
	getMetrics = func(m *MesosStats, ip string) map[string]float64 {
		return map[string]float64{
			"test": 0.1,
		}
	}

	c := make(chan metric.Metric)
	sut := newMesosStats(c, 10, defaultLog).(*MesosStats)

	go sut.sendMetrics()
	actual := <-c

	assert.Equal(t, expected, actual)
}

func TestMesosStatsGetMetrics(t *testing.T) {
	oldGetMetricsURL := getMetricsURL
	defer func() {
		getMetricsURL = oldGetMetricsURL
	}()

	tests := []struct {
		rawResponse string
		expected    map[string]float64
		msg         string
	}{
		{"{\"frameworks\\/chronos\\/messages_processed\":6784068, \"master\\/elected\": 1}", map[string]float64{"frameworks.chronos.messages_processed": 6784068, "master.elected": 1}, "Valid JSON should return valid metrics."},
		{"{\"frameworks\\/chronos\\/messages_processed\":6784068, \"master\\/elected\": 0}", map[string]float64{}, "Only the elected master should return metrics."},
		{"{\"frameworks\\/chronos\\/messages_processed6784068}", nil, "Invalid JSON should return nil."},
	}

	for _, test := range tests {
		expected := test.expected
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, test.rawResponse)
		}))
		defer ts.Close()

		getMetricsURL = func(ip string) string { return ts.URL }

		sut := newMesosStats(nil, 10, defaultLog).(*MesosStats)
		actual := getMetrics(sut, httptest.DefaultRemoteAddr)

		assert.Equal(t, expected, actual)
	}
}

func TestMesosStatsGetMetricsHandleErrors(t *testing.T) {
	oldGetMetricsURL := getMetricsURL
	defer func() {
		getMetricsURL = oldGetMetricsURL
	}()

	getMetricsURL = func(ip string) string { return "" }

	sut := newMesosStats(nil, 10, defaultLog).(*MesosStats)
	actual := getMetrics(sut, httptest.DefaultRemoteAddr)

	assert.Nil(t, actual, "Empty (invalid) URL, which means http client should throw an error; therefore, we expect a nil from getMetrics")
}

func TestMesosStatsGetMetricsHandleNon200s(t *testing.T) {
	oldGetMetricsURL := getMetricsURL
	defer func() {
		getMetricsURL = oldGetMetricsURL
	}()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Custom error`)
	}))
	defer ts.Close()

	getMetricsURL = func(ip string) string { return ts.URL }

	sut := newMesosStats(nil, 10, defaultLog).(*MesosStats)
	actual := getMetrics(sut, httptest.DefaultRemoteAddr)

	assert.Nil(t, actual, "Server threw a 500, so we should expect nil from getMetrics")
}

func TestMesosStatsBuildMetric(t *testing.T) {
	expected := metric.Metric{"mesos.test", "gauge", 0.1, map[string]string{}}

	actual := buildMetric("test", 0.1)

	assert.Equal(t, expected, actual)
}

func TestMesosStatsBuildMetricCumCounter(t *testing.T) {
	expected := metric.Metric{"mesos.master.slave_reregistrations", metric.CumulativeCounter, 0.1, map[string]string{}}

	actual := buildMetric("master.slave_reregistrations", 0.1)

	assert.Equal(t, expected, actual)
}

package config_test

import (
	"fullerite/config"

	"io/ioutil"
	"os"
	"testing"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var testBadConfiguration = `{
    "prefix": "test.",
    malformed JSON File {123!!!!
}
`

var testGoodConfiguration = `{
    "prefix": "test.",
    "interval": 10,
    "defaultDimensions": {
        "application": "fullerite",
        "host": "dev33-devc"
    },

    "collectorsConfigPath": "/tmp",
    "diamondCollectorsPath": "src/diamond/collectors",
    "diamondCollectors": ["CPUCollector","PingCollector"],
    "collectors": ["Test"],

    "handlers": {
        "Graphite": {
            "server": "10.40.11.51",
            "port": "2003",
            "timeout": 2
        },
        "SignalFx": {
            "authToken": "secret_token",
            "endpoint": "https://ingest.signalfx.com/v2/datapoint",
            "interval": 10,
            "timeout": 2,
			"collectorBlackList": ["TestCollector1", "TestCollector2"]
        }
    }
}
`

var testCollectorConfiguration = `{
	"metricName": "TestMetric",
	"interval": %%COLLECTOR_INTERVAL%%
}
`

var testGoodEnvvarConfiguration = `{
		"prefix": "%%PREFIX%%",
		"interval": 10,
		"defaultDimensions": {
				"application": "fullerite",
				"this_should_interpolate": "%%THIS_SHOULD_INTERPOLATE%%",
				"test_percent1": "%IS_NOT_AFFECTED",
				"test_percent2": "%IS_NOT_AFFECTED%",
				"test_percent3": "%%IS_NOT_AFFECTED%",
				"test_percent4": "%%IS_ALSO_NOT_AFFECTED_BECAUSE_NOT_SET%%",
				"test_percent4": "%%IS_ALSO_NOT_AFFECTED_BECAUSE_NOT_SET%%",
				"test_short_does_not_work1": "asijwef%%TS1%%iwefwewef",
				"test_short_does_not_work2": "asijwef%%TS_1%%iwefwewef",
				"test_numeric_start_does_not_work": "%%1MISSISSIPPI%%",
				"test_lowercase_does_not_work": "%%this_is_lowercase%%",
				"host": "dev33-devc"
		},
		"collectorsConfigPath": "/tmp",
		"diamondCollectorsPath": "src/diamond/collectors",
		"diamondCollectors": ["CPUCollector","PingCollector"],
		"collectors": ["Test"],
		"handlers": {
				"SignalFx": {
						"authToken": "%%SIGNALFX_AUTH_TOKEN%%",
						"endpoint": "https://ingest.signalfx.com/v2/datapoint",
						"interval": 10,
						"timeout": 2,
			"collectorBlackList": ["TestCollector1", "TestCollector2"]
			}
		}
}`

var (
	tmpTestGoodFile, tmpTestGoodEnvvarFile, tmpTestBadFile, tempTestCollectorConfig string
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.ErrorLevel)
	if f, err := ioutil.TempFile("/tmp", "fullerite"); err == nil {
		f.WriteString(testGoodConfiguration)
		tmpTestGoodFile = f.Name()
		f.Close()
		defer os.Remove(tmpTestGoodFile)
	}
	if f, err := ioutil.TempFile("/tmp", "fullerite"); err == nil {
		f.WriteString(testBadConfiguration)
		tmpTestBadFile = f.Name()
		f.Close()
		defer os.Remove(tmpTestBadFile)
	}
	if f, err := ioutil.TempFile("/tmp", "fullerite"); err == nil {
		f.WriteString(testGoodEnvvarConfiguration)
		tmpTestGoodEnvvarFile = f.Name()
		f.Close()
		defer os.Remove(tmpTestBadFile)
	}
	if f, err := ioutil.TempFile("/tmp", "fullerite"); err == nil {
		f.WriteString(testCollectorConfiguration)
		tempTestCollectorConfig = f.Name()
		f.Close()
		defer os.Remove(tempTestCollectorConfig)
	}
	os.Exit(m.Run())
}

func TestGetInt(t *testing.T) {
	assert := assert.New(t)

	val := config.GetAsInt("10", 123)
	assert.Equal(val, 10)

	val = config.GetAsInt("notanint", 123)
	assert.Equal(val, 123)

	val = config.GetAsInt(12.123, 123)
	assert.Equal(val, 12)

	val = config.GetAsInt(12, 123)
	assert.Equal(val, 12)
}

func TestGetFloat(t *testing.T) {
	assert := assert.New(t)

	val := config.GetAsFloat("10", 123)
	assert.Equal(val, 10.0)

	val = config.GetAsFloat("10.21", 123)
	assert.Equal(val, 10.21)

	val = config.GetAsFloat("notanint", 123)
	assert.Equal(val, 123.0)

	val = config.GetAsFloat(12.123, 123)
	assert.Equal(val, 12.123)
}

func TestGetAsMap(t *testing.T) {
	assert := assert.New(t)

	// Test if string can be converted to map[string]string
	stringToParse := "{\"runtimeenv\" : \"dev\", \"region\":\"uswest1-devc\"}"
	expectedValue := map[string]string{
		"runtimeenv": "dev",
		"region":     "uswest1-devc",
	}
	assert.Equal(config.GetAsMap(stringToParse), expectedValue)

	// Test if map[string]interface{} can be converted to map[string]string
	interfaceMapToParse := make(map[string]interface{})
	interfaceMapToParse["runtimeenv"] = "dev"
	interfaceMapToParse["region"] = "uswest1-devc"
	assert.Equal(config.GetAsMap(interfaceMapToParse), expectedValue)
}

func TestGetAsSlice(t *testing.T) {
	assert := assert.New(t)

	// Test if string array can be converted to []string
	stringToParse := "[\"TestCollector1\", \"TestCollector2\"]"
	expectedValue := []string{"TestCollector1", "TestCollector2"}
	assert.Equal(config.GetAsSlice(stringToParse), expectedValue)

	sliceToParse := []string{"TestCollector1", "TestCollector2"}
	assert.Equal(config.GetAsSlice(sliceToParse), expectedValue)
}

func TestGetAsSliceFromJson(t *testing.T) {
	var data interface{}
	jsonString := []byte(`{"listOfStrings": ["a", "b", "c"]}`)

	err := json.Unmarshal(jsonString, &data)
	assert.Nil(t, err)

	if err == nil {
		temp := data.(map[string]interface{})

		res := config.GetAsSlice(temp["listOfStrings"])
		assert.Equal(t, []string{"a", "b", "c"}, res)
	}
}

func TestParseCollectorConfig(t *testing.T) {
	_ = os.Setenv("COLLECTOR_INTERVAL", "10")
	ret, err := config.ReadCollectorConfig(tempTestCollectorConfig)
	assert.Nil(t, err, "should succeed")
	assert.Equal(t,
		10.0,
		ret["interval"],
		"COLLECTOR_INTERVAL should be interpolated correctly",
	)
}

func TestParseGoodConfig(t *testing.T) {
	_, err := config.ReadConfig(tmpTestGoodFile)
	assert.Nil(t, err, "should succeed")
}

func TestParseBadConfig(t *testing.T) {
	_, err := config.ReadConfig(tmpTestBadFile)
	assert.NotNil(t, err, "should fail")
}

func TestParseGoodEnvvarConfig(t *testing.T) {
	prefixEnv := "prefix"
	_ = os.Setenv("PREFIX", prefixEnv)
	authToken := "blah_blah_auth_blah"
	_ = os.Setenv("SIGNALFX_AUTH_TOKEN", authToken)
	isNotAffected := "this should not appear"
	_ = os.Setenv("IS_NOT_AFFECTED", isNotAffected)
	shouldInterpolate := "yey this got interpolated"
	_ = os.Setenv("THIS_SHOULD_INTERPOLATE", shouldInterpolate)
	tooShortEnvvar := "We do not want to accidental short values, eg into qwef1%%TS1%%wwskfe"
	_ = os.Setenv("TS1", tooShortEnvvar)
	_ = os.Setenv("TS_1", tooShortEnvvar)
	noStartWithNumeric := "Why would you want a number at start of envvar"
	_ = os.Setenv("1MISSISSIPPI", noStartWithNumeric)
	mustBeUppercase := "By convention, envvars are generally uppercase+numeric+underscore"
	_ = os.Setenv("this_is_lowercase", mustBeUppercase)

	ret, err := config.ReadConfig(tmpTestGoodEnvvarFile)

	assert.Nil(t, err, "should succeed")
	assert.Equal(t,
		prefixEnv,
		ret.Prefix,
		"prefix should be interpolated",
	)
	assert.Equal(t,
		authToken,
		ret.Handlers["SignalFx"]["authToken"],
		"handler token should be interpolated",
	)
	assert.Equal(t,
		shouldInterpolate,
		ret.DefaultDimensions["this_should_interpolate"],
		"dimension should be interpolated",
	)
	assert.Equal(t,
		"%IS_NOT_AFFECTED",
		ret.DefaultDimensions["test_percent1"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"%IS_NOT_AFFECTED%",
		ret.DefaultDimensions["test_percent2"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"%%IS_NOT_AFFECTED%",
		ret.DefaultDimensions["test_percent3"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"%%IS_ALSO_NOT_AFFECTED_BECAUSE_NOT_SET%%",
		ret.DefaultDimensions["test_percent4"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"asijwef%%TS1%%iwefwewef",
		ret.DefaultDimensions["test_short_does_not_work1"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"asijwef%%TS_1%%iwefwewef",
		ret.DefaultDimensions["test_short_does_not_work2"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"%%1MISSISSIPPI%%",
		ret.DefaultDimensions["test_numeric_start_does_not_work"],
		"should not be interpolated",
	)
	assert.Equal(t,
		"%%this_is_lowercase%%",
		ret.DefaultDimensions["test_lowercase_does_not_work"],
		"should not be interpolated",
	)
}

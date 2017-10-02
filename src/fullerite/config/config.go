package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
)

var log = logrus.WithFields(logrus.Fields{"app": "fullerite", "pkg": "config"})

// Config type holds the global Fullerite configuration.
type Config struct {
	Prefix                string                            `json:"prefix"`
	Interval              interface{}                       `json:"interval"`
	CollectorsConfigPath  string                            `json:"collectorsConfigPath"`
	DiamondCollectorsPath string                            `json:"diamondCollectorsPath"`
	DiamondCollectors     []string                          `json:"diamondCollectors"`
	Handlers              map[string]map[string]interface{} `json:"handlers"`
	Collectors            []string                          `json:"collectors"`
	DefaultDimensions     map[string]string                 `json:"defaultDimensions"`
	InternalServerConfig  map[string]interface{}            `json:"internalServer"`
}

// interpolateEnvvarsIntoConfigRead replaces any values like %%MY_ENVVAR%% with their
// corresponding environment variable. Limits to min 5 chars, uppercase, must start with
// letter. Can include underscore and numerics.
func interpolateEnvvarsIntoConfigRead(configFile string) ([]byte, error) {
	c, e := ioutil.ReadFile(configFile)
	re, _ := regexp.Compile("%%[A-Z][A-Z0-9_]{4,}%%")
	matched := re.FindAll(c, -1)
	for _, v := range matched {
		potentialEnvvar := string(v)[2 : len(v)-2] // slice off the lead/trail %%'s
		if val, ok := os.LookupEnv(potentialEnvvar); ok {
			re2, _ := regexp.Compile("%%" + potentialEnvvar + "%%")
			c = re2.ReplaceAllLiteral(c, []byte(val))
		}
	}
	return c, e
}

// ReadConfig reads a fullerite configuration file
func ReadConfig(configFile string) (c Config, e error) {
	log.Info("Reading configuration file at ", configFile)
	contents, e := interpolateEnvvarsIntoConfigRead(configFile)
	if e != nil {
		log.Error("Config file error: ", e)
		return c, e
	}
	err := json.Unmarshal(contents, &c)
	if err != nil {
		log.Error("Invalid JSON in config: ", err)
		return c, err
	}
	return c, nil
}

// ReadCollectorConfig reads a fullerite collector configuration file
func ReadCollectorConfig(configFile string) (c map[string]interface{}, e error) {
	log.Info("Reading collector configuration file at ", configFile)
	contents, e := interpolateEnvvarsIntoConfigRead(configFile)
	if e != nil {
		log.Error("Config file error: ", e)
		return c, e
	}
	err := json.Unmarshal(contents, &c)
	if err != nil {
		log.Error("Invalid JSON in config: ", err)
		return c, err
	}
	return c, nil
}

// GetCollectorConfig returns collector config. given a name
func (conf Config) GetCollectorConfig(name string) (map[string]interface{}, error) {
	configFile := strings.Join([]string{conf.CollectorsConfigPath, name}, "/") + ".conf"
	// Since collector naems can be defined with a space in order to instantiate multiple
	// instances of the same collector, we want their files
	// will not have that space and needs to have it replaced with an underscore
	// instead
	configFile = strings.Replace(configFile, " ", "_", -1)
	collectorConf, err := ReadCollectorConfig(configFile)
	return collectorConf, err
}

// GetAsFloat parses a string to a float or returns the float if float is passed in
func GetAsFloat(value interface{}, defaultValue float64) (result float64) {
	result = defaultValue

	switch value.(type) {
	case string:
		fromString, err := strconv.ParseFloat(value.(string), 64)
		if err != nil {
			log.Warn("Failed to convert value", value, "to a float64. Falling back to default", defaultValue)
			result = defaultValue
		} else {
			result = fromString
		}
	case float64:
		result = value.(float64)
	}

	return
}

// GetAsInt parses a string/float to an int or returns the int if int is passed in
func GetAsInt(value interface{}, defaultValue int) (result int) {
	result = defaultValue

	switch value.(type) {
	case string:
		fromString, err := strconv.ParseInt(value.(string), 10, 64)
		if err == nil {
			result = int(fromString)
		} else {
			log.Warn("Failed to convert value", value, "to an int")
		}
	case int:
		result = value.(int)
	case int32:
		result = int(value.(int32))
	case int64:
		result = int(value.(int64))
	case float64:
		result = int(value.(float64))
	}

	return
}

// GetAsMap parses a string to a map[string]string
func GetAsMap(value interface{}) (result map[string]string) {
	result = make(map[string]string)

	switch value.(type) {
	case string:
		err := json.Unmarshal([]byte(value.(string)), &result)
		if err != nil {
			log.Warn("Failed to convert value", value, "to a map")
		}
	case map[string]interface{}:
		temp := value.(map[string]interface{})
		for k, v := range temp {
			if str, ok := v.(string); ok {
				result[k] = str
			} else {
				log.Warn("Expected a string but got", reflect.TypeOf(value), ". Discarding handler level metric: ", k)
			}
		}
	case map[string]string:
		result = value.(map[string]string)
	default:
		log.Warn("Expected a string but got", reflect.TypeOf(value), ". Returning empty map!")
	}

	return
}

// GetAsSlice : Parses a json array string to []string
func GetAsSlice(value interface{}) []string {
	result := []string{}

	switch realValue := value.(type) {
	case string:
		err := json.Unmarshal([]byte(realValue), &result)
		if err != nil {
			log.Warn("Failed to convert string:", realValue, "to a []string")
		}
	case []string:
		result = realValue
	case []interface{}:
		result = make([]string, len(realValue))
		for i, value := range realValue {
			result[i] = value.(string)
		}
	default:
		log.Warn("Expected a string array but got", reflect.TypeOf(realValue), ". Returning empty slice!")
	}

	return result
}

package config

import (
	"encoding/json"
	"fmt"
)

type (
	Opts struct {
		// logger
		Logger struct {
			Debug   bool `           long:"debug"        env:"DEBUG"    description:"debug mode"`
			Verbose bool `short:"v"  long:"verbose"      env:"VERBOSE"  description:"verbose mode"`
			LogJson bool `           long:"log.json"     env:"LOG_JSON" description:"Switch log output to json format"`
		}

		// OpenSearch
		OpenSearch struct {
			// OpenSearch settings
			Addresses []string `long:"opensearch.address"    env:"OPENSEARCH_ADDRESS"    delim:" "   default:"http://localhost:9200"   description:"OpenSearch urls" required:"true"`
			Username  string   `long:"opensearch.username"   env:"OPENSEARCH_USERNAME"                                                 description:"OpenSearch username for HTTP Basic Authentication"`
			Password  string   `long:"opensearch.password"   env:"OPENSEARCH_PASSWORD"                                                 description:"OpenSearch password for HTTP Basic Authentication" json:"-"`
			Index     string   `long:"opensearch.index"      env:"OPENSEARCH_INDEX"                  default:"alertmanager-%y.%m"      description:"OpenSearch index name (placeholders: %y for year, %m for month and %d for day)"`
		}

		// general options
		ServerBind string `long:"bind"     env:"SERVER_BIND"   description:"Server address"     default:":9097"`
	}
)

func (o *Opts) GetJson() (string, error) {
	jsonBytes, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("marshal err: %w", err)
	}
	return string(jsonBytes), nil
}

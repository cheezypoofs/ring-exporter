package exporter

import (
	"encoding/json"
	"github.com/cheezypoofs/ring-exporter/ringapi"
	"github.com/pkg/errors"
	"io/ioutil"
)

// WebConfig contains the serializable config items for the web service.
type WebConfig struct {
	Port         uint32 `json:"port"`
	MetricsRoute string `json:"metrics_route"`
}

// EnsureWebConfigDefaults handles setting sane defaults
// and migrating the config forward. It returns `true` if
// any changes were made.
func EnsureWebConfigDefaults(config *WebConfig) bool {
	dirty := false
	if config.Port == 0 {
		dirty = true
		config.Port = 9100
	}
	if config.MetricsRoute == "" {
		dirty = true
		config.MetricsRoute = "/metrics"
	}
	return dirty
}

type Config struct {
	ApiConfig ringapi.ApiConfig `json:"api_config"`
	WebConfig WebConfig         `json:"web_config"`

	PollIntervalSeconds uint32 `json:"poll_interval_seconds"`
	SaveIntervalSeconds uint32 `json:"save_interval_seconds"`
}

func EnsureConfigDefaults(cfg *Config) bool {
	dirty := false
	if cfg.PollIntervalSeconds == 0 {
		dirty = true
		cfg.PollIntervalSeconds = 5 * 60
	}
	if cfg.SaveIntervalSeconds == 0 {
		dirty = true
		cfg.SaveIntervalSeconds = 5 * 60
	}
	if ringapi.EnsureApiConfigDefaults(&cfg.ApiConfig) {
		dirty = true
	}
	if EnsureWebConfigDefaults(&cfg.WebConfig) {
		dirty = true
	}
	return dirty
}

func LoadConfig(filename string) (*Config, error) {
	cfg := &Config{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read config")
	}

	if json.Unmarshal(data, cfg); err != nil {
		return nil, errors.Wrapf(err, "Failed to deserialize config")
	}

	if EnsureConfigDefaults(cfg) {
		SaveConfig(filename, cfg)
	}

	return cfg, nil
}

func SaveConfig(filename string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		// unexpected
		panic(err)
	}

	if err = ioutil.WriteFile(filename, data, 0600); err != nil {
		return errors.Wrapf(err, "Failed to persist config")
	}
	return nil
}

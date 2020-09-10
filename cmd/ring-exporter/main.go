package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cheezypoofs/ring-exporter/ringapi"
	ring_types "github.com/cheezypoofs/ring-exporter/ringapi/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func sanitizeLabelValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "_")
	s = strings.ReplaceAll(s, "\r", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

//////////////////
type simpleFileTokenHandler struct {
	filename string
}

func newSimpleFileTokenHandler(cfgFile string) *simpleFileTokenHandler {
	tokenFile := filepath.Join(filepath.Dir(cfgFile), "ring-token.json")

	return &simpleFileTokenHandler{
		filename: tokenFile,
	}
}

func (s *simpleFileTokenHandler) Fetch() *oauth2.Token {
	token := &oauth2.Token{}
	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		return nil
	}

	if json.Unmarshal(data, token); err != nil {
		return nil
	}

	return token
}

func (s *simpleFileTokenHandler) Store(token *oauth2.Token) {
	data, _ := json.MarshalIndent(token, "", " ")
	ioutil.WriteFile(s.filename, data, 0600)
}

////////////////

type cliAuthenticator struct {
}

func (*cliAuthenticator) PromptCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	password := string(bytePassword)
	fmt.Printf("\n")

	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func (*cliAuthenticator) Prompt2FACode() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter 2FA Code: ")
	code, _ := reader.ReadString('\n')
	return strings.TrimSpace(code), nil
}

/////////////////////

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

/////////////////////

type config struct {
	ApiConfig ringapi.ApiConfig `json:"api_config"`
	WebConfig WebConfig         `json:"web_config"`

	PollIntervalSeconds uint32 `json:"poll_interval_seconds"`
}

func ensureConfigDefaults(cfg *config) bool {
	dirty := false
	if cfg.PollIntervalSeconds == 0 {
		dirty = true
		cfg.PollIntervalSeconds = 5 * 60
	}
	if ringapi.EnsureApiConfigDefaults(&cfg.ApiConfig) {
		dirty = true
	}
	if EnsureWebConfigDefaults(&cfg.WebConfig) {
		dirty = true
	}
	return dirty
}

func loadConfig(filename string) (*config, error) {
	cfg := &config{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read config")
	}

	if json.Unmarshal(data, cfg); err != nil {
		return nil, errors.Wrapf(err, "Failed to deserialize config")
	}

	if ensureConfigDefaults(cfg) {
		saveConfig(filename, cfg)
	}

	return cfg, nil
}

func saveConfig(filename string, cfg *config) error {
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

//////////////
// command handlers
//////////////

func handleInit(cfgFile string) error {

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		// Let's initialize a new config

		cfg = &config{}

		if ensureConfigDefaults(cfg) {
			if err = saveConfig(cfgFile, cfg); err != nil {
				return err
			}
		}
	}

	// Now, let's authenticate a new token
	if _, err = ringapi.OpenAuthorizedSession(cfg.ApiConfig, newSimpleFileTokenHandler(cfgFile), &cliAuthenticator{}); err != nil {
		return errors.Wrapf(err, "Failed to authorize new token")
	}

	return nil
}

func handleTest(cfgFile string) error {

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	// No authenticator. Should fail if we don't have a token.
	session, err := ringapi.OpenAuthorizedSession(cfg.ApiConfig, newSimpleFileTokenHandler(cfgFile), nil)
	if err != nil {
		return err
	}

	// Let's test the token.
	info, err := session.GetSessionInfo()
	if err != nil {
		return errors.Wrapf(err, "Failed to obtain session info")
	}

	fmt.Printf("Ready to work with your bells, %s %s\n", info.Profile.FirstName, info.Profile.LastName)

	return nil
}

func handleMonitor(cfgFile string, metrics *prometheus.Registry, quitter chan struct{}) error {

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	session, err := ringapi.OpenAuthorizedSession(cfg.ApiConfig, newSimpleFileTokenHandler(cfgFile), nil)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)

	batteryLevel := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ring_device_battery_pct",
		Help: "Device battery level (percent)",
	}, []string{
		"description",
		"type",
	})
	metrics.MustRegister(batteryLevel)

	wifiSignal := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ring_device_wifi_strength_dbm",
		Help: "Latest wifi strength reading (-dBm)",
	}, []string{
		"description",
		"type",
	})
	metrics.MustRegister(wifiSignal)

	updateDeviceMetrics := func(description string, health *ring_types.DeviceHealth, typ string) {
		// Let's get battery level
		if health.BatteryPercentage != nil {
			bl := batteryLevel.With(prometheus.Labels{
				"description": sanitizeLabelValue(description),
				"type":        typ,
			})
			f, err := strconv.ParseFloat(*health.BatteryPercentage, 64)
			if err != nil {
				log.Printf("Skipping %s due to failure parsing battery pct '%s'", description, health.BatteryPercentage)
				bl.Set(math.NaN())
			} else {
				log.Printf("Device %s has battery pct %f", description, f)
				bl.Set(f)
			}
		}

		// And some wifi stats
		if health.LatestSignalStrength != nil {
			ws := wifiSignal.With(prometheus.Labels{
				"description": sanitizeLabelValue(description),
				"type":        typ,
			})
			ws.Set(float64(*health.LatestSignalStrength))
		}
	}

	pollOnce := func() {
		devices, err := session.GetDevices()
		if err != nil {
			log.Printf("Failed to retrieve device info: %v", err)
			return
		}

		for _, device := range devices.DoorBots {

			// Get the health. It has more details
			hr, err := session.GetDoorBotHealth(&device)
			if err != nil {
				log.Printf("Skipping %s because of failed health fetch: %v", device.Description, err)
				continue
			}
			updateDeviceMetrics(device.Description, &hr.DeviceHealth, "doorbot")
		}

		for _, device := range devices.Chimes {

			// Get the health. It has more details
			cr, err := session.GetChimeHealth(&device)
			if err != nil {
				log.Printf("Skipping %s because of failed health fetch: %v", device.Description, err)
				continue
			}
			updateDeviceMetrics(device.Description, &cr.DeviceHealth, "chime")
		}
	}

	pollOnce()

	go func() {
		for {
			select {
			case <-ticker.C:
				pollOnce()
			case <-quitter:
				ticker.Stop()
				return
			}
		}
	}()

	go func() {
		http.Handle(cfg.WebConfig.MetricsRoute, promhttp.HandlerFor(metrics, promhttp.HandlerOpts{}))
		http.ListenAndServe(fmt.Sprintf(":%d", cfg.WebConfig.Port), nil)
	}()

	return nil
}

///////////////////////

func main() {

	cfg := struct {
		configFile string
	}{}

	a := kingpin.New(filepath.Base(os.Args[0]), "A Prometheus exporter for Ring devices")
	a.HelpFlag.Short('h')

	a.Command("init", "Initialize (or reinitialize) for background usage")
	a.Command("test", "Test the configuration and token")
	a.Command("monitor", "Execute monitoring and exposition of metrics")

	a.Flag("config.file", "JSON configuration file").
		Default("ring-config.json").StringVar(&cfg.configFile)

	parsed, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	quitter := make(chan struct{})
	metrics := prometheus.NewRegistry()

	switch parsed {
	case "init":
		err = handleInit(cfg.configFile)
	case "test":
		err = handleTest(cfg.configFile)
	case "monitor":
		err = handleMonitor(cfg.configFile, metrics, quitter)

		select {
		case <-quitter:
		}
	default:
		err = fmt.Errorf("oops")
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

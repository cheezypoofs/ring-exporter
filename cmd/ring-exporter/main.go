package main

import (
	"fmt"
	"github.com/cheezypoofs/ring-exporter/exporter"
	"github.com/cheezypoofs/ring-exporter/ringapi"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

//////////////
// command handlers
//////////////

func handleInit(cfgFile string) error {

	cfg, err := exporter.LoadConfig(cfgFile)
	if err != nil {
		// Let's initialize a new config

		cfg = &exporter.Config{}

		if exporter.EnsureConfigDefaults(cfg) {
			if err = exporter.SaveConfig(cfgFile, cfg); err != nil {
				return err
			}
		}
	}

	// Now, let's authenticate a new token
	if _, err = ringapi.OpenAuthorizedSession(cfg.ApiConfig, exporter.NewRingStateHandler(cfgFile), &exporter.CliAuthenticator{}); err != nil {
		return errors.Wrapf(err, "Failed to authorize new token")
	}

	return nil
}

func handleTest(cfgFile string) error {

	cfg, err := exporter.LoadConfig(cfgFile)
	if err != nil {
		return err
	}

	// No authenticator. Should fail if we don't have a token.
	session, err := ringapi.OpenAuthorizedSession(cfg.ApiConfig, exporter.NewRingStateHandler(cfgFile), nil)
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

	monitor, err := exporter.NewMonitor(cfgFile, metrics)
	if err != nil {
		return err
	}

	monitor.PollOnce()
	pollTicker := time.NewTicker(time.Duration(monitor.Config.PollIntervalSeconds) * time.Second)
	saveTicker := time.NewTicker(time.Duration(monitor.Config.SaveIntervalSeconds) * time.Second)

	go func() {
		for {
			select {
			case <-pollTicker.C:
				monitor.PollOnce()
			case <-saveTicker.C:
				monitor.StateHandler.Save()
			case <-quitter:
				pollTicker.Stop()
				saveTicker.Stop()
				return
			}
		}
	}()

	go func() {
		http.Handle(monitor.Config.WebConfig.MetricsRoute, promhttp.HandlerFor(metrics, promhttp.HandlerOpts{}))
		http.ListenAndServe(fmt.Sprintf(":%d", monitor.Config.WebConfig.Port), nil)
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

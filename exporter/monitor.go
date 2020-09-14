package exporter

import (
	"github.com/cheezypoofs/ring-exporter/ringapi"
	ring_types "github.com/cheezypoofs/ring-exporter/ringapi/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"math"
	"strconv"
	"strings"
)

const (
	doorbotType      = "dootbot"
	chimeType        = "chime"
	descriptionLabel = "description"
	typeLabel        = "type"
)

func sanitizeLabelValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "_")
	s = strings.ReplaceAll(s, "\r", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

// Monitor performs the ringapi query calls and population into prometheus metrics
type Monitor struct {
	StateHandler *RingStateHandler
	Session      *ringapi.AuthorizedSession
	Config       *Config

	batteryLevel *prometheus.GaugeVec
	wifiSignal   *prometheus.GaugeVec
	dingsCount   *prometheus.GaugeVec
}

// NewMonitor creates a new Monitor instance with the required parameters
func NewMonitor(cfgFile string, metrics *prometheus.Registry) (*Monitor, error) {

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		return nil, err
	}

	stateHandler := NewRingStateHandler(cfgFile)

	session, err := ringapi.OpenAuthorizedSession(cfg.ApiConfig, stateHandler, nil)
	if err != nil {
		return nil, err
	}

	monitor := &Monitor{
		StateHandler: stateHandler,
		Session:      session,
		Config:       cfg,
		batteryLevel: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ring_device_battery_pct",
			Help: "Device battery level (percent)",
		}, []string{
			descriptionLabel,
			typeLabel,
		}),
		wifiSignal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ring_device_wifi_strength_dbm",
			Help: "Latest wifi strength reading (-dBm)",
		}, []string{
			descriptionLabel,
			typeLabel,
		}),
		// It's a counter, but we're explicitly sampling a value we best-effort count and persist ourselves
		// so it's really more like a gauge of a counter we don't control.
		dingsCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ring_device_dings_total",
			Help: "Best-effort count of total dings",
		}, []string{
			descriptionLabel,
			typeLabel,
		}),
	}

	metrics.MustRegister(monitor.batteryLevel)
	metrics.MustRegister(monitor.wifiSignal)
	metrics.MustRegister(monitor.dingsCount)

	return monitor, nil
}

func (m *Monitor) updateDingMetrics(device *ring_types.DoorBot, dings *[]ring_types.DoorBotDing) {
	curCount, err := m.StateHandler.UpdateDingCount(device, dings)

	if err != nil {
		return
	}

	m.dingsCount.With(prometheus.Labels{
		"description": sanitizeLabelValue(device.Description),
		"type":        doorbotType,
	}).Set(float64(curCount))
	log.Printf("Device %s has current ding count %d", device.Description, curCount)
}

func (m *Monitor) updateDeviceMetrics(description string, health *ring_types.DeviceHealth, typ string) {

	// Let's get battery level
	if health.BatteryPercentage != nil {
		bl := m.batteryLevel.With(prometheus.Labels{
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
		ws := m.wifiSignal.With(prometheus.Labels{
			"description": sanitizeLabelValue(description),
			"type":        typ,
		})
		log.Printf("Device %s has wifi strength %f", description, *health.LatestSignalStrength)
		ws.Set(float64(*health.LatestSignalStrength))
	}
}

// PollOnce performs the API queries and metrics updates
func (m *Monitor) PollOnce() error {

	devices, err := m.Session.GetDevices()
	if err != nil {
		return errors.Wrapf(err, "Failed to retrieve device info")
	}

	for _, device := range devices.DoorBots {

		// Get the health. It has more details
		hr, err := m.Session.GetDoorBotHealth(&device)
		if err != nil {
			log.Printf("Skipping %s because of failed health fetch: %v", device.Description, err)
			continue
		}
		m.updateDeviceMetrics(device.Description, &hr.DeviceHealth, doorbotType)

		dings, err := m.Session.GetDoorBotHistory(&device)
		if err == nil {
			m.updateDingMetrics(&device, &dings)
		}
	}

	for _, device := range devices.Chimes {

		// Get the health. It has more details
		cr, err := m.Session.GetChimeHealth(&device)
		if err != nil {
			log.Printf("Skipping %s because of failed health fetch: %v", device.Description, err)
			continue
		}

		m.updateDeviceMetrics(device.Description, &cr.DeviceHealth, chimeType)
	}

	return nil
}

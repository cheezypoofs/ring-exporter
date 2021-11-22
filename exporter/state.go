package exporter

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/cheezypoofs/ring-exporter/ringapi"
	ring_types "github.com/cheezypoofs/ring-exporter/ringapi/types"
	"golang.org/x/oauth2"
)

type dingCount struct {
	DeviceId      uint32    `json:"device_id"`
	MyCounter     uint32    `json:"my_counter"`
	LastTimestamp time.Time `json:"last_timestamp"`
}

// RingState is a serializable object for holding state
type RingState struct {
	Token      *oauth2.Token `json:"token"`
	DingCounts []dingCount   `json:"ding_counts"`
}

// RingStateHandler exposes persistence of the `RingState` and also implements
// `ringapi.TokenHandler`
type RingStateHandler struct {
	filename string

	lock  sync.Mutex
	state RingState
}

// NewRingStateHandler creates a new RingStateHandler instance
func NewRingStateHandler(cfgFile string) *RingStateHandler {
	stateFile := filepath.Join(filepath.Dir(cfgFile), "ring-state.json")

	handler := &RingStateHandler{
		filename: stateFile,
	}
	handler.load()
	return handler
}

// Fetch implements `ringapi.TokenHandler` interface
func (s *RingStateHandler) FetchToken() *oauth2.Token {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.state.Token
}

// Fetch implements `ringapi.TokenHandler` interface
func (s *RingStateHandler) StoreToken(token *oauth2.Token) {
	s.lock.Lock()
	s.state.Token = token
	s.saveLocked()
	s.lock.Unlock()
}

func (s *RingStateHandler) load() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		return err
	}

	if json.Unmarshal(data, &s.state); err != nil {
		return err
	}

	return nil
}

func (s *RingStateHandler) Save() {
	s.lock.Lock()
	s.saveLocked()
	s.lock.Unlock()
}

func (s *RingStateHandler) saveLocked() {
	data, _ := json.MarshalIndent(s.state, "", " ")
	ioutil.WriteFile(s.filename, data, 0600)
}

// UpdateDingCount updates the `RingState` with the historical set of dings and updates
// the gauge reprsenting the ding counter appropriate. This returns the current count (across restart)
// for the device as seen by this exporter and state.
func (s *RingStateHandler) UpdateDingCount(id uint32, dings *[]ring_types.DoorBotDing) (uint32, error) {

	s.lock.Lock()
	defer s.lock.Unlock()

	var count *dingCount

	// See if we have state already for this device
	for _, d := range s.state.DingCounts {
		if d.DeviceId == id {
			// grab a reference
			count = &d
			break
		}
	}

	// Create a new state entry for this device
	if count == nil {
		newCount := dingCount{
			DeviceId: id,
		}
		s.state.DingCounts = append(s.state.DingCounts, newCount)
		// grab a reference to it
		count = &s.state.DingCounts[len(s.state.DingCounts)-1]
	}

	// Keep track of the latest bookmark we have
	lastTimestamp := count.LastTimestamp

	for _, ding := range *dings {
		ts, err := time.Parse(time.RFC3339, ding.CreatedAt)
		if err != nil {
			continue
		}

		// Count this ding if it's after the last bookmark
		if ts.After(count.LastTimestamp) {
			count.MyCounter++
		}

		// Move our bookmark forward to the most recent one we've seen
		if ts.After(lastTimestamp) {
			lastTimestamp = ts
		}
	}

	// Persist the bookmark
	count.LastTimestamp = lastTimestamp

	// And let the caller know the current count for this device now
	return count.MyCounter, nil
}

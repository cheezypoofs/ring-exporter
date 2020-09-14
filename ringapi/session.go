package ringapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	ring_types "github.com/cheezypoofs/ring-exporter/ringapi/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

///////////////////////////////////

// AuthorizedSession is necessary to perform ring API calls. Retreive this instance
// by calling OpenAuthorizedSession.
type AuthorizedSession struct {
	client *http.Client
	config ApiConfig
}

func query(client *http.Client, method string, uri string, inParam io.Reader, outParam interface{}) error {
	request, err := http.NewRequest(method, baseUrl+uri, inParam)
	if err != nil {
		return err
	}

	query := request.URL.Query()
	query.Add("api_version", apiVersion)

	request.URL.RawQuery = query.Encode()

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(b)

	// useful for dumping payloads in development
	// fmt.Println(body)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("API request failed %v %v", resp.Status, body)
	}

	decoder := json.NewDecoder(body)

	// note: to remain compatible with additions, don't use decoder.DisallowUnknownFields()

	if err = decoder.Decode(outParam); err != nil {
		return err
	}

	return nil
}

// GetSessionInfo fetches information about the current API session
func (session *AuthorizedSession) GetSessionInfo() (*ring_types.SessionResponse, error) {
	var loginForm = url.Values{
		"api_version":                            {apiVersion},
		"device[hardware_id]":                    {session.config.HardwareId},
		"device[os]":                             {"android"},
		"device[app_brand]":                      {"ring"},
		"device[metadata][device_model]":         {""},
		"device[metadata][device_name]":          {""},
		"device[metadata][resolution]":           {""},
		"device[metadata][app_version]":          {""},
		"device[metadata][app_instalation_date]": {""},
		"device[metadata][manufacturer]":         {""},
		"device[metadata][device_type]":          {"desktop"},
		"device[metadata][architecture]":         {""},
		"device[metadata][language]":             {"en"},
	}

	sessionResponse := &ring_types.SessionResponse{}
	if err := query(session.client, "POST", uriSession, strings.NewReader(loginForm.Encode()), sessionResponse); err != nil {
		return nil, err
	}

	return sessionResponse, nil
}

// GetDevices fetches the ring devices in the current API session.
func (session *AuthorizedSession) GetDevices() (*ring_types.DevicesResponse, error) {
	devicesRespsonse := &ring_types.DevicesResponse{}
	if err := query(session.client, "GET", uriRingDevices, nil, devicesRespsonse); err != nil {
		return nil, err
	}

	return devicesRespsonse, nil
}

// GetDoorBotHealth fetches the health info for a particular id.
func (session *AuthorizedSession) GetDoorBotHealth(bot *ring_types.DoorBot) (*ring_types.DoorBotHealthResponse, error) {
	healthResponse := &ring_types.DoorBotHealthResponse{}
	if err := query(session.client, "GET", fmt.Sprintf(uriDoorbots, bot.Id)+uriHealth, nil, healthResponse); err != nil {
		return nil, err
	}

	return healthResponse, nil
}

// GetChimeHealth fetches the health info for a particular id.
func (session *AuthorizedSession) GetChimeHealth(chime *ring_types.Chime) (*ring_types.DoorBotHealthResponse, error) {
	healthResponse := &ring_types.DoorBotHealthResponse{}
	if err := query(session.client, "GET", fmt.Sprintf(uriChimes, chime.Id)+uriHealth, nil, healthResponse); err != nil {
		return nil, err
	}

	return healthResponse, nil
}

func (session *AuthorizedSession) GetDoorBotHistory(bot *ring_types.DoorBot) ([]ring_types.DoorBotDing, error) {
	var response []ring_types.DoorBotDing
	if err := query(session.client, "GET", fmt.Sprintf(uriDoorbots, bot.Id)+uriHistory, nil, &response); err != nil {
		return nil, err
	}

	return response, nil
}

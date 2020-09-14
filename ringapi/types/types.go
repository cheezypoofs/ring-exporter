package types

///////////////////////////////////
// Data structures gleaned from projects referenced in the README
// and manual inspection of API payloads experimentally.
//////////////////////////////////

// Features exposes features from the ring session API.
type Features struct {
	RemoteLoggingFormatStoring               bool   `json:"remote_logging_format_storing"`
	RemoteLoggingLevel                       int    `json:"remote_logging_level"`
	SubscriptionsEnabled                     bool   `json:"subscriptions_enabled"`
	StickupcamSetupEnabled                   bool   `json:"stickupcam_setup_enabled"`
	VodEnabled                               bool   `json:"vod_enabled"`
	RingplusEnabled                          bool   `json:"ringplus_enabled"`
	LpdEnabled                               bool   `json:"lpd_enabled"`
	ReactiveSnoozingEnabled                  bool   `json:"reactive_snoozing_enabled"`
	ProactiveSnoozingEnabled                 bool   `json:"proactive_snoozing_enabled"`
	OwnerProactiveSnoozingEnabled            bool   `json:"owner_proactive_snoozing_enabled"`
	LiveViewSettingsEnabled                  bool   `json:"live_view_settings_enabled"`
	DeleteAllSettingsEnabled                 bool   `json:"delete_all_settings_enabled"`
	PowerCableEnabled                        bool   `json:"power_cable_enabled"`
	DeviceHealthAlertsEnabled                bool   `json:"device_health_alerts_enabled"`
	ChimeProEnabled                          bool   `json:"chime_pro_enabled"`
	MultipleCallsEnabled                     bool   `json:"multiple_calls_enabled"`
	UjetEnabled                              bool   `json:"ujet_enabled"`
	MultipleDeleteEnabled                    bool   `json:"multiple_delete_enabled"`
	DeleteAllEnabled                         bool   `json:"delete_all_enabled"`
	LpdMotionAnnouncementEnabled             bool   `json:"lpd_motion_announcement_enabled"`
	StarredEventsEnabled                     bool   `json:"starred_events_enabled"`
	ChimeDndEnabled                          bool   `json:"chime_dnd_enabled"`
	VideoSearchEnabled                       bool   `json:"video_search_enabled"`
	FloodlightCamEnabled                     bool   `json:"floodlight_cam_enabled"`
	RingCamBatteryEnabled                    bool   `json:"ring_cam_battery_enabled"`
	EliteCamEnabled                          bool   `json:"elite_cam_enabled"`
	DoorbellV2Enabled                        bool   `json:"doorbell_v2_enabled"`
	SpotlightBatteryDashboardControlsEnabled bool   `json:"spotlight_battery_dashboard_controls_enabled"`
	BypassAccountVerification                bool   `json:"bypass_account_verification"`
	LegacyCvrRetentionEnabled                bool   `json:"legacy_cvr_retention_enabled"`
	RingCamEnabled                           bool   `json:"ring_cam_enabled"`
	RingSearchEnabled                        bool   `json:"ring_search_enabled"`
	RingCamMountEnabled                      bool   `json:"ring_cam_mount_enabled"`
	RingAlarmEnabled                         bool   `json:"ring_alarm_enabled"`
	InAppCallNotifications                   bool   `json:"in_app_call_notifications"`
	RingCashEligibleEnabled                  bool   `json:"ring_cash_eligible_enabled"`
	AppAlertTonesEnabled                     bool   `json:"app_alert_tones_enabled"`
	MotionSnoozingEnabled                    bool   `json:"motion_snoozing_enabled"`
	HistoryClassificationEnabled             bool   `json:"history_classification_enabled"`
	TileDashboardEnabled                     bool   `json:"tile_dashboard_enabled"`
	TileDashboardMode                        string `json:"tile_dashboard_mode"`
	ScrubberAutoLiveEnabled                  bool   `json:"scrubber_auto_live_enabled"`
	ScrubberEnabled                          bool   `json:"scrubber_enabled"`
	NwEnabled                                bool   `json:"nw_enabled"`
	NwV2Enabled                              bool   `json:"nw_v2_enabled"`
	NwFeedTypesEnabled                       bool   `json:"nw_feed_types_enabled"`
	NwLargerAreaEnabled                      bool   `json:"nw_larger_area_enabled"`
	NwUserActivated                          bool   `json:"nw_user_activated"`
	NwNotificationTypesEnabled               bool   `json:"nw_notification_types_enabled"`
	NwNotificationRadiusEnabled              bool   `json:"nw_notification_radius_enabled"`
	NwMapViewFeatureEnabled                  bool   `json:"nw_map_view_feature_enabled"`
}

// Profile exposes the user profile info from the ring session API.
type Profile struct {
	Id                   int      `json:"id"`
	Email                string   `json:"email"`
	FirstName            string   `json:"first_name"`
	LastName             string   `json:"last_name"`
	PhoneNumber          string   `json:"phone_number"`
	AuthenticationToken  string   `json:"authentication_token"`
	HardwareId           string   `json:"hardware_id"`
	ExplorerProgramTerms string   `json:"explorer_program_terms"`
	UserFlow             string   `json:"user_flow"`
	AppBrand             string   `json:"app_brand"`
	Features             Features `json:"features"`
}

// SessionResponse exposes is the top-level response from ring session API.
type SessionResponse struct {
	Profile Profile `json:"profile"`
}

// DoorBot is an instance of a device on a door returned from the ring devices API.
type DoorBot struct {
	Id          uint32  `json:"id"`
	Description string  `json:"description"`
	DeviceId    string  `json:"device_id"`
	BatteryLife *string `json:"battery_life"`
	// note: There are many other data available
}

type Chime struct {
	Id          uint32 `json:"id"`
	Description string `json:"description"`
	// note: There are many other data available
}

// DevicesResponse is the top-level response from the ring devices API.
type DevicesResponse struct {
	DoorBots []DoorBot `json:"doorbots"`
	Chimes   []Chime   `json:"chimes"`
}

// DeviceHealth describes a single DoorBot's health
type DeviceHealth struct {
	Id       uint32  `json:"id"`
	WifiName *string `json:"wifi_name"`
	// BatteryPercentage can be null and is a string, not a float so you need to convert.
	BatteryPercentage         *string `json:"battery_percentage"`
	BatteryPercentageCategory *string `json:"battery_percentage_category"`
	// note: I don't use these and I'm unsure what they look like as mine were null.
	//	BatteryVoltage interface{} `json:"battery_voltage"`
	//	BatteryVoltageCategory interface{} `json:"battery_voltage_category"`
	LatestSignalStrength  *float32 `json:"latest_signal_strength"`
	LatestSignalCategory  *string  `json:"latest_signal_category"`
	AverageSignalStregnth *float32 `json:"average_signal_strength"`
	AverageSignalCategory *string  `json:"average_signal_category"`
	Firmware              string   `json:"firmware"`
	UpdatedAt             string   `json:"updated_at"`
	WifiIsRingNetwork     bool     `json:"wifi_is_ring_network"`
	//	PacketLossCategory string `json:"packet_loss_category"`
	//	PacketLossStrength interface{} `json:"packet_loss_strength"`
	//	ExternalPowerState uint32 `json:"ext_power_state"`
}

// ChimeHealthResponse is the top-level response from the ring doorbot health API query.
type DoorBotHealthResponse struct {
	DeviceHealth DeviceHealth `json:"device_health"`
}

// ChimeHealthResponse is the top-level response from the ring chime health API query.
type ChimeHealthResponse struct {
	DeviceHealth DeviceHealth `json:"device_health"`
}

type DoorBotDing struct {
	Id        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
	Kind      string `json:"kind"`
}

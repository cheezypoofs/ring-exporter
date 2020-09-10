package ringapi

import (
	"github.com/google/uuid"
)

// ApiConfig provides a way to alter API behaviors. This is a serializable
// structure.
type ApiConfig struct {
	HardwareId string `json:"hardware_id"`
}

// EnsureApiConfigDefaults handles setting sane defaults
// and migrating the config forward. It returns `true` if
// any changes were made.
func EnsureApiConfigDefaults(config *ApiConfig) bool {
	dirty := false

	if config.HardwareId == "" {
		dirty = true
		hardwareId, _ := uuid.NewRandom()
		config.HardwareId = hardwareId.String()
	}
	return dirty
}

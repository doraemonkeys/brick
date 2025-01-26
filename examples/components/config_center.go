package components

import (
	"encoding/json"

	"github.com/doraemonkeys/brick"
)

// ConfigCenter implements Brick and provides configuration management.
type ConfigCenter struct {
	Config map[string]string
}

// BrickTypeID implements the Brick interface.
func (*ConfigCenter) BrickTypeID() string {
	return "ConfigCenter"
}

// NewBrick implements the BrickNewer interface.
func (cc *ConfigCenter) NewBrick(jsonConfig []byte) brick.Brick {
	config := make(map[string]string)
	if len(jsonConfig) > 0 {
		if err := json.Unmarshal(jsonConfig, &config); err != nil {
			panic(err)
		}
	}
	return &ConfigCenter{Config: config}
}

// GetConfig retrieves a configuration value by key.
func (cc *ConfigCenter) GetConfig(key string) string {
	if cc.Config == nil {
		return ""
	}
	return cc.Config[key]
}

package components

import (
	"encoding/json"
	"fmt"

	"github.com/doraemonkeys/brick"
)

// CacheService implements Brick and provides caching functionality.
type CacheService struct {
	Config map[string]string
}

// BrickTypeID implements the Brick interface.
func (*CacheService) BrickTypeID() string {
	return "CacheService"
}

// NewBrick implements the BrickNewer interface.
func (cs *CacheService) NewBrick(jsonConfig []byte) brick.Brick {
	config := make(map[string]string)
	if len(jsonConfig) > 0 {
		if err := json.Unmarshal(jsonConfig, &config); err != nil {
			panic(err)
		}
	}
	return &CacheService{Config: config}
}

// GetCache retrieves a value from the cache (simulated).
func (cs *CacheService) GetCache(key string) string {
	fmt.Printf("Retrieving data from cache with key: %s\n", key)
	if cs.Config == nil {
		return ""
	}
	return cs.Config[key]
}

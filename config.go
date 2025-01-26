package brick

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"gopkg.in/yaml.v3"
)

// AddConfigFile adds brick configurations from a file, supporting JSON and YAML formats.
func AddConfigFile(path string) error {
	return brickManager.AddConfigFile(path)
}

// AddConfigFile adds brick configurations from a file, supporting JSON and YAML formats.
func (b *BrickManager) AddConfigFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return b.addConfigFileJson(content)
	case ".yaml", ".yml":
		return b.addConfigFileYaml(content)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
}

// addConfig adds brick configurations from a slice of BrickFileConfig.
func (b *BrickManager) addConfig(configs []BrickFileConfig) error {

	liveIDMap := make(map[string]bool)
	for _, config := range configs {
		if config.MetaData.TypeID == "" {
			return fmt.Errorf("typeID is required")
		}
		singleTypeliveIDs := make(map[string]bool, len(config.Lives))
		for _, live := range config.Lives {
			if live.LiveID == "" {
				return fmt.Errorf("the liveID of brick(%s) is required", config.MetaData.TypeID)
			}
			if _, ok := liveIDMap[live.LiveID]; ok {
				return fmt.Errorf("liveID duplicate: %s", live.LiveID)
			}
			liveIDMap[live.LiveID] = true
			singleTypeliveIDs[live.LiveID] = true
		}
		if b.liveIDConstraint && len(singleTypeliveIDs) > 0 && !singleTypeliveIDs[config.MetaData.TypeID] {
			return fmt.Errorf("the liveID of all instances of the brick must have one set to the typeID(%s) of the brick", config.MetaData.TypeID)
		}
	}

	b.brickConfigLock.RLock()
	for _, config := range configs {
		for _, live := range config.Lives {
			if _, ok := b.brickConfigs[live.LiveID]; ok {
				return fmt.Errorf("liveID duplicate: %s", live.LiveID)
			}
		}
	}
	b.brickConfigLock.RUnlock()

	for _, config := range configs {
		checked := false
		if config.MetaData.NoCheck {
			checked = true
		}
		for _, live := range config.Lives {
			if !checked && live.Config != nil {
				if _, ok := b.getBrickType(config.MetaData.TypeID); ok {
					if _, ok = b.getBrickFactory(config.MetaData.TypeID); !ok {
						return fmt.Errorf("the brick(%s) provides config, but no config parser, please use `brick.RegisterNewer` to register the brick", config.MetaData.TypeID)
					}
				}
				checked = true
			}
			if live.LiveID != config.MetaData.TypeID {
				b.setDeclaredLiveID(live.LiveID)
			}
			b.setBrickConfig(live.LiveID, BrickConfig{
				TypeID:  config.MetaData.TypeID,
				LiveID:  live.LiveID,
				Config:  live.Config,
				noCheck: config.MetaData.NoCheck,
			})
		}
	}
	// reset
	b.brickConfigCheckOnce = sync.Once{}
	return nil
}

func (b *BrickManager) checkConfig() {
	b.brickConfigLock.RLock()
	for _, config := range b.brickConfigs {
		if config.Config != nil && !config.noCheck {
			// if _, ok := b.getBrickType(config.TypeID); !ok {
			// 	panic(fmt.Errorf("the typeID(%s) of brick is not registered", config.TypeID))
			// }
			if _, ok := b.getBrickFactory(config.TypeID); ok {
				continue
			}
			typ, ok := b.getBrickType(config.TypeID)
			if !ok {
				panic(fmt.Errorf("the brick(%s) provided config, but it is not registered.\nPlease use `brick.RegisterNewer` to register the brick, or set `noCheck: true` in the config file", config.TypeID))
			} else {
				panic(fmt.Errorf("the brick(%s) provided config, but no config parser. typeID(%s)\nPlease use `brick.RegisterNewer` to register the brick, or set `noCheck: true` in the config file", typ, config.TypeID))
			}
		}
	}
	b.brickConfigLock.RUnlock()
}

// addConfigFileYaml adds brick configurations from YAML content.
func (b *BrickManager) addConfigFileYaml(yamlContent []byte) error {
	var configs1 struct {
		Bricks []BrickFileConfig `yaml:"bricks"`
	}
	err := yaml.Unmarshal(yamlContent, &configs1)
	if err == nil {
		return b.addConfig(configs1.Bricks)
	}
	var configs2 []BrickFileConfig
	err2 := yaml.Unmarshal(yamlContent, &configs2)
	if err2 == nil {
		return b.addConfig(configs2)
	}
	return errors.New("invalid config file format")
}

// addConfigFileJson adds brick configurations from JSON content.
func (b *BrickManager) addConfigFileJson(jsonContent []byte) error {
	var configs1 struct {
		Bricks []BrickFileConfig `json:"bricks"`
	}
	err1 := json.Unmarshal(jsonContent, &configs1)
	if err1 == nil && configs1.Bricks != nil {
		return b.addConfig(configs1.Bricks)
	}
	var configs2 []BrickFileConfig
	err2 := json.Unmarshal(jsonContent, &configs2)
	if err2 == nil {
		return b.addConfig(configs2)
	}
	return errors.New("invalid config file format")
}

// saveBrickInstance saves a created brick instance.
func (b *BrickManager) saveBrickInstance(liveID string, brick reflect.Value) {
	b.instancesLock.Lock()
	defer b.instancesLock.Unlock()
	b.instances[liveID] = brick
}

// getBrickFromExist retrieves an existing brick instance by LiveID.
func (b *BrickManager) getBrickFromExist(liveID string) (reflect.Value, bool) {
	b.instancesLock.RLock()
	defer b.instancesLock.RUnlock()
	brick, ok := b.instances[liveID]
	return brick, ok
}

// getBrickConfig retrieves a brick's configuration by LiveID.
func (b *BrickManager) getBrickConfig(liveID string) (BrickConfig, bool) {
	b.brickConfigLock.RLock()
	defer b.brickConfigLock.RUnlock()
	brickConfig, ok := b.brickConfigs[liveID]
	return brickConfig, ok
}

// setBrickConfig sets a brick's configuration by LiveID.
func (b *BrickManager) setBrickConfig(liveID string, brickConfig BrickConfig) {
	brickConfig.LiveID = liveID
	b.brickConfigLock.Lock()
	defer b.brickConfigLock.Unlock()
	b.brickConfigs[liveID] = brickConfig
}

// setBrickBuilding sets the building status of a brick by LiveID.
func (b *BrickManager) setBrickBuilding(liveID string, building bool) {
	b.buildingBrickLock.Lock()
	defer b.buildingBrickLock.Unlock()
	b.buildingBrick[liveID] = building
}

// getBrickBuilding gets the building status of a brick by LiveID.
func (b *BrickManager) getBrickBuilding(liveID string) bool {
	b.buildingBrickLock.Lock()
	defer b.buildingBrickLock.Unlock()
	return b.buildingBrick[liveID]
}

func (b *BrickManager) setBrickTypeID(typ reflect.Type, typeID string) (success bool) {
	b.brickTypeIDMapLock.Lock()
	defer b.brickTypeIDMapLock.Unlock()
	if _, ok := b.brickTypeIDMap1[typ]; ok {
		return false
	}
	// fmt.Println("setBrickTypeID", typ, typeID)
	b.brickTypeIDMap1[typ] = typeID
	b.brickTypeIDMap2[typeID] = typ
	return true
}
func (b *BrickManager) getBrickTypeID(typ reflect.Type) (string, bool) {
	b.brickTypeIDMapLock.RLock()
	defer b.brickTypeIDMapLock.RUnlock()
	typeID, ok := b.brickTypeIDMap1[typ]
	return typeID, ok
}

func (b *BrickManager) getBrickType(typeID string) (reflect.Type, bool) {
	b.brickTypeIDMapLock.RLock()
	defer b.brickTypeIDMapLock.RUnlock()
	typ, ok := b.brickTypeIDMap2[typeID]
	return typ, ok
}

func (b *BrickManager) getBrickFactory(typeID string) (func(config any) Brick, bool) {
	b.brickFactoriesLock.RLock()
	defer b.brickFactoriesLock.RUnlock()
	factory, ok := b.brickFactories[typeID]
	return factory, ok
}

// func (b *BrickManager) setBrickFactory(typeID string, factory func(config any) Brick) {
// 	b.brickFactoriesLock.Lock()
// 	defer b.brickFactoriesLock.Unlock()
// 	b.brickFactories[typeID] = factory
// }

func (b *BrickManager) getDeclaredLiveID(liveID string) bool {
	b.declaredLiveIDsLock.RLock()
	defer b.declaredLiveIDsLock.RUnlock()
	return b.declaredLiveIDs[liveID]
}

func (b *BrickManager) setDeclaredLiveID(liveID string) {
	b.declaredLiveIDsLock.Lock()
	defer b.declaredLiveIDsLock.Unlock()
	b.declaredLiveIDs[liveID] = true
}

type TypeLives struct {
	TypeID string
	Lives  []Live
}

type Live struct {
	// TypeID string
	LiveID string
	// key:field name, value:liveID
	RelyLives map[string]string
}

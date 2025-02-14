package brick

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	defer func() {
		b.configsLock.Lock()
		for i := 0; i < len(b.configs); i++ {
			if b.configs[i].filePath == path {
				panic(fmt.Errorf("config file(%s) already exists", path))
			}
		}
		b.configs = append(b.configs, NewConfigManager(path))
		b.configsLock.Unlock()
	}()
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

// handleConfig replaces environment variables in a configuration.
func handleConfig(config any) any {
	c, _ := handleConfigHelper(config)
	return c
}

// handleConfigHelper is a recursive helper function for handleConfig.
func handleConfigHelper(config any) (conf any, maybeReplaced bool) {
	switch val := config.(type) {
	case string:
		if isEnvConfigItem(val) {
			conf, _ = handleConfigHelper(os.ExpandEnv(val))
			return conf, true
		}
	case map[string]any:
		for k, v := range val {
			c, replaced := handleConfigHelper(v)
			if replaced {
				val[k] = c
			}
		}
	case map[string]string:
		for k, v := range val {
			c, replaced := handleConfigHelper(v)
			if replaced {
				val[k] = c.(string)
			}
		}
	case []any:
		for i, v := range val {
			c, replaced := handleConfigHelper(v)
			if replaced {
				val[i] = c
			}
		}
	case []string:
		for i, v := range val {
			c, replaced := handleConfigHelper(v)
			if replaced {
				val[i] = c.(string)
			}
		}
	}
	return config, false
}

// saveBrickInstance saves a created brick instance.
func (b *BrickManager) saveBrickInstance(liveID string, brick reflect.Value) {
	if brick.Type().Kind() != reflect.Ptr {
		panic(fmt.Errorf("internal error: brick(%s) instance is not a pointer", liveID))
	}
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

func (b *BrickManager) saveBrickConfig(typeID string, brickLiveID string, brickConfig []byte) error {
	var err error
	for i := 0; i < len(b.configs); i++ {
		b.configsLock.RLock()
		config := b.configs[i]
		b.configsLock.RUnlock()
		if err = config.saveBrickConfig(typeID, brickLiveID, brickConfig); err == nil {
			return nil
		}
	}
	return err
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

type BrickBase[T Brick] struct {
	// brickTypeID string
	// BrickLiveID string
	liveID string `json:"-" yaml:"-" toml:"-"`
	// lock        *sync.Mutex
}

func (c BrickBase[T]) SaveBrickConfig(config any) error {
	j, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return brickManager.saveBrickConfig(GetBrickTypeID[T](), c.liveID, j)
}

func (c BrickBase[T]) BrickLiveID() string {
	if c.liveID == "" {
		panic(fmt.Errorf("this brick(%s) has not been injected", GetBrickTypeID[T]()))
	}
	return c.liveID
}

func (c BrickBase[T]) NewBrick(config []byte) Brick {
	var newBrick = *new(T)
	if len(config) > 0 {
		if err := json.Unmarshal(config, newBrick); err != nil {
			panic(fmt.Errorf("parse brick config error: %w", err))
		}
		return newBrick
	}
	typ := reflect.TypeOf(newBrick)
	if typ.Kind() != reflect.Ptr {
		return newBrick
	}
	typ = typ.Elem()
	emptyInstance := createEmptyPtrInstance(typ)
	return emptyInstance.Interface().(Brick)
}

type ConfigManager struct {
	configMu sync.Mutex
	// used to determine if the config has changed
	// lastLoadedConfig []byte
	configIsArray bool
	filePath      string
}

func NewConfigManager(filePath string) *ConfigManager {
	var configManager = ConfigManager{
		filePath: filePath,
	}
	return &configManager
}

func (c *ConfigManager) Load() ([]BrickFileConfig, error) {
	content, err := os.ReadFile(c.filePath)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(c.filePath)
	switch ext {
	case ".json":
		return c.loadJson(content)
	// case ".yaml", ".yml":
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func (c *ConfigManager) loadJson(content []byte) ([]BrickFileConfig, error) {
	var configs1 struct {
		Bricks []BrickFileConfig `json:"bricks"`
	}
	err1 := json.Unmarshal(content, &configs1)
	if err1 == nil && configs1.Bricks != nil {
		// brickBytes, err := json.Marshal(configs1.Bricks)
		// if err != nil {
		// 	return nil, err
		// }
		// c.lastLoadedConfig = brickBytes
		return configs1.Bricks, nil
	}
	var configs2 []BrickFileConfig
	err2 := json.Unmarshal(content, &configs2)
	if err2 == nil {
		// c.lastLoadedConfig = content
		c.configIsArray = true
		return configs2, nil
	}
	return nil, errors.New("invalid config file format")
}

func (c *ConfigManager) saveBrickConfig(typeID string, brickLiveID string, brickConfig []byte) (err error) {
	// oldLastLoadedConfig := c.lastLoadedConfig
	configs, err := c.Load()
	if err != nil {
		return err
	}
	var brickConfigParsed any
	err = json.Unmarshal(brickConfig, &brickConfigParsed)
	if err != nil {
		return err
	}
	// configIndex := -1
	for i, config := range configs {
		if config.MetaData.TypeID == typeID {
			for j, live := range config.Lives {
				if live.LiveID == brickLiveID {
					newEnvs := make(map[string]string)
					configs[i].Lives[j].Config, _ = retainEnvConfigItem(configs[i].Lives[j].Config, brickConfigParsed, newEnvs)
					c.configMu.Lock()
					defer c.configMu.Unlock()
					return c.saveChanged(brickLiveID, configs, i, j, newEnvs)
				}
			}
		}
	}
	return fmt.Errorf("brick(%s) config not found", brickLiveID)
	// if configIndex == -1 {
	// 	var newConfig BrickFileConfig
	// 	newConfig.MetaData.TypeID = typeID
	// 	configs = append(configs, newConfig)
	// 	configIndex = len(configs) - 1
	// }
	// var newLive = struct {
	// 	LiveID string `json:"liveID" yaml:"liveID" toml:"liveID"`
	// 	Config any    `json:"config" yaml:"config" toml:"config"`
	// }{
	// 	LiveID: brickLiveID,
	// 	Config: brickConfig,
	// }
	// newLive.LiveID = brickLiveID
	// newLive.Config = brickConfigParsed
	// configs[configIndex].Lives = append(configs[configIndex].Lives, newLive)
	// return c.saveToFile(configs)
}

func (c *ConfigManager) saveChanged(brickLiveID string, configs []BrickFileConfig, i, j int, newEnvs map[string]string) error {
	var err2 error
	ext := filepath.Ext(c.filePath)
	switch ext {
	case ".json":
		if c.configIsArray {
			content, err := json.MarshalIndent(configs, "", "    ")
			if err != nil {
				return err
			}
			err2 = WriteFilePerm(c.filePath, content)
		} else {
			fileContent, err := os.ReadFile(c.filePath)
			if err != nil {
				return err
			}
			var allConfigs map[string]any
			err = json.Unmarshal(fileContent, &allConfigs)
			if err != nil {
				return err
			}
			allConfigs["bricks"] = configs
			content, err := json.MarshalIndent(allConfigs, "", "    ")
			if err != nil {
				return err
			}
			err2 = WriteFilePerm(c.filePath, content)
		}
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
	if err2 != nil {
		return err2
	}
	for k, v := range newEnvs {
		setEnvConfigItem(k, v)
	}
	brickManager.setBrickConfig(brickLiveID, BrickConfig{
		TypeID:  configs[i].MetaData.TypeID,
		LiveID:  brickLiveID,
		noCheck: configs[i].MetaData.NoCheck,
		Config:  configs[i].Lives[j].Config,
	})
	return nil
}

func isEnvConfigItem(item string) bool {
	return strings.HasPrefix(item, "${") && strings.HasSuffix(item, "}")
}

func setEnvConfigItem(item string, value string) {
	item = strings.TrimPrefix(item, "${")
	item = strings.TrimSuffix(item, "}")
	_ = os.Setenv(item, value)
}

func retainEnvConfigItem(oldConfig any, newConfig any, newEnvs map[string]string) (conf any, maybeReplaced bool) {
	switch val := oldConfig.(type) {
	case string:
		newVal, ok := newConfig.(string)
		if !ok {
			return newConfig, false
		}
		if isEnvConfigItem(val) {
			// setEnvConfigItem(val, newVal)
			newEnvs[val] = newVal
			return oldConfig, true
		}
	case map[string]any:
		newVal, ok := newConfig.(map[string]any)
		if !ok {
			return newConfig, false
		}
		for k, v := range val {
			c, replaced := retainEnvConfigItem(v, newVal[k], newEnvs)
			if replaced {
				newVal[k] = c
			}
		}
	case map[string]string:
		newVal, ok := newConfig.(map[string]string)
		if !ok {
			return newConfig, false
		}
		for k, v := range val {
			c, replaced := retainEnvConfigItem(v, newVal[k], newEnvs)
			if replaced {
				newVal[k] = c.(string)
			}
		}
	case []any:
		newVal, ok := newConfig.([]any)
		if !ok {
			return newConfig, false
		}
		for i, v := range val {
			c, replaced := retainEnvConfigItem(v, newVal[i], newEnvs)
			if replaced {
				newVal[i] = c
			}
		}
	case []string:
		newVal, ok := newConfig.([]string)
		if !ok {
			return newConfig, false
		}
		for i, v := range val {
			c, replaced := retainEnvConfigItem(v, newVal[i], newEnvs)
			if replaced {
				newVal[i] = c.(string)
			}
		}
	}
	return newConfig, false
}

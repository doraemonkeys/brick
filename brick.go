package brick

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
)

var (
	brickManager = &BrickManager{
		brickConfigs:     make(map[string]BrickConfig),
		instances:        make(map[string]reflect.Value),
		buildingBrick:    make(map[string]bool),
		brickFactories:   make(map[string]func(config any) Brick),
		brickTypeIDMap1:  make(map[reflect.Type]string),
		brickTypeIDMap2:  make(map[string]reflect.Type),
		liveIDTypeMap:    make(map[string]reflect.Type),
		declaredLiveIDs:  make(map[string]bool),
		liveIDConstraint: true,
	}
)

const brickTag = "brick"

// BrickManager manages brick configurations and instances.
type BrickManager struct {
	// brickConfigs stores configurations for each brick, indexed by LiveID.
	brickConfigs    map[string]BrickConfig
	brickConfigLock sync.RWMutex

	// instances stores created brick instances, indexed by LiveID.
	// All instances are saved as pointers.
	instances     map[string]reflect.Value
	instancesLock sync.RWMutex

	// brickFactories stores functions to parse configurations into bricks, indexed by TypeID.
	// If the Brick interface is implemented by a value receiver, the type returned by the function may be a value type or a pointer type.
	brickFactories     map[string]func(config any) Brick
	brickFactoriesLock sync.RWMutex

	// brickTypeIDMap1 stores the TypeID of each brick type, indexed by reflect.Type.
	brickTypeIDMap1    map[reflect.Type]string
	brickTypeIDMap2    map[string]reflect.Type
	brickTypeIDMapLock sync.RWMutex

	// liveIDTypeMap is a map that stores the type of registered liveID, indexed by liveID.
	liveIDTypeMap     map[string]reflect.Type
	liveIDTypeMapLock sync.RWMutex

	// All liveIDs declared in the configuration file or tag
	declaredLiveIDs     map[string]bool
	declaredLiveIDsLock sync.RWMutex

	// buildingBrick tracks if a brick is currently being built, to prevent circular dependencies,
	// indexed by LiveID.
	buildingBrick     map[string]bool
	buildingBrickLock sync.Mutex

	brickConfigCheckOnce sync.Once

	// liveIDConstraint is a flag to control whether the constraint that all instances of the same brick type must have one liveID set to typeID is enabled.
	liveIDConstraint bool
}

// BrickConfig holds the configuration for a single brick instance.
type BrickConfig struct {
	TypeID string
	LiveID string
	// If a brick type provides configuration but does not implement the BrickNewer interface,
	// an error will be thrown by default.
	// NoCheck is a flag to control whether the configuration should be checked.
	noCheck bool
	Config  any
}

// BrickFileConfig defines the structure of a brick configuration file.
type BrickFileConfig struct {
	MetaData struct {
		Name    string `json:"name" yaml:"name" toml:"name"`
		TypeID  string `json:"typeID" yaml:"typeID" toml:"typeID"`
		NoCheck bool   `json:"noCheck" yaml:"noCheck" toml:"noCheck"`
	} `json:"metaData" yaml:"metaData" toml:"metaData"`
	Lives []struct {
		LiveID string `json:"liveID" yaml:"liveID" toml:"liveID"`
		Config any    `json:"config" yaml:"config" toml:"config"`
	} `json:"lives" yaml:"lives" toml:"lives"`
}

// Brick is the interface that all bricks must implement.
type Brick interface {
	// BrickTypeID returns a unique constant string for each brick type.
	BrickTypeID() string
}

type BrickNewer interface {
	Brick
	// NewBrick parses the configuration and returns a new instance of the brick.
	// If the configuration file does not provide the configuration for the component, jsonConfig is nil.
	//
	// To ensure that this method can be called even if receiver is nil, please create a new instance and return it.
	NewBrick(jsonConfig []byte) Brick
}

type BrickLives interface {
	BrickNewer
	// Used when multiple different instances of a type depend on different instances of the same type.
	// The dependency relationship returned here will forcibly override the dependency relationship in the tag.
	BrickLives() []Live
}

var brickInterfaceType = reflect.TypeOf((*Brick)(nil)).Elem()
var brickNewerInterfaceType = reflect.TypeOf((*BrickNewer)(nil)).Elem()
var brickLivesInterfaceType = reflect.TypeOf((*BrickLives)(nil)).Elem()

// func generateBrickLivesInterfaceFromType(typ reflect.Type) BrickLives {

// }

func getBrickLives(typ reflect.Type) ([]Live, bool) {
	if typ.Kind() != reflect.Ptr {
		if typ.Implements(brickLivesInterfaceType) {
			instance := reflect.New(typ).Interface().(BrickLives)
			return instance.BrickLives(), true
		}
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	typPtr := reflect.PointerTo(typ)
	if !typPtr.Implements(brickLivesInterfaceType) {
		return nil, false
	}
	instance := reflect.New(typ).Interface().(BrickLives)
	return instance.BrickLives(), true
}

// SetLiveIDConstraint sets the constraint that all instances of the same brick type must have one liveID set to typeID.
func SetLiveIDConstraint(constraint bool) {
	brickManager.liveIDConstraint = constraint
}

// Register registers the brick type and recursively registers its all dependencies.
// It uses the BrickTypeID method of the provided type to determine the type ID.
// This method is used for bricks that do not require custom configuration parsing.
func Register[T Brick]() {
	// Pointer receiver registers pointer type, value receiver registers value type
	var instance = *new(T)
	var typ = reflect.TypeOf(instance)
	if typ.Kind() != reflect.Ptr {
		brickManager.register2(instance.BrickTypeID(), typ)
		return
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	newInstancePtr := reflect.New(typ)
	if typ.Implements(brickInterfaceType) {
		brick, _ := newInstancePtr.Elem().Interface().(Brick)
		brickManager.register2(brick.BrickTypeID(), typ)
		return
	}
	brick, _ := newInstancePtr.Interface().(Brick)
	brickManager.register2(brick.BrickTypeID(), newInstancePtr.Type())
}

// RegisterNewer like Register, but it also registers a factory for configuration parsing.
// The provided type must implement the BrickNewer interface, which includes the NewBrick method
// for parsing configurations.
func RegisterNewer[T BrickNewer]() {
	// Pointer receiver registers pointer type, value receiver registers value type
	var instance = *new(T)
	var typ = reflect.TypeOf(instance)
	if typ.Kind() != reflect.Ptr {
		brickManager.register2(instance.BrickTypeID(), typ, instance.NewBrick)
		return
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	newInstancePtr := reflect.New(typ)
	if typ.Implements(brickNewerInterfaceType) {
		brick, _ := newInstancePtr.Elem().Interface().(BrickNewer)
		brickManager.register2(brick.BrickTypeID(), typ, brick.NewBrick)
		return
	}
	brick, _ := newInstancePtr.Interface().(BrickNewer)
	brickManager.register2(brick.BrickTypeID(), newInstancePtr.Type(), brick.NewBrick)
}

func RegisterLives[T BrickLives]() {
	var instance = *new(T)
	var typ = reflect.TypeOf(instance)
	var param = RegisterBrickParam{}
	defer func() {
		for _, live := range param.Lives {
			brickManager.setDeclaredLiveID(live.LiveID)
			for _, depLive := range live.RelyLives {
				brickManager.setDeclaredLiveID(depLive)
			}
		}
		brickManager.register(param)
	}()
	if typ.Kind() != reflect.Ptr {
		param.ReflectType = typ
		param.TypeID = instance.BrickTypeID()
		param.Lives = instance.BrickLives()
		param.BrickFactory = instance.NewBrick
		return
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	newInstancePtr := reflect.New(typ)
	if typ.Implements(brickLivesInterfaceType) {
		brick, _ := newInstancePtr.Elem().Interface().(BrickLives)
		param.TypeID = brick.BrickTypeID()
		param.Lives = brick.BrickLives()
		param.ReflectType = typ
		param.BrickFactory = brick.NewBrick
		return
	}
	brick, _ := newInstancePtr.Interface().(BrickLives)
	param.TypeID = brick.BrickTypeID()
	param.Lives = brick.BrickLives()
	param.ReflectType = typ
	param.BrickFactory = brick.NewBrick
}

func (b *BrickManager) RegisterLiveIDType(liveID string, reflectType reflect.Type) {
	b.liveIDTypeMapLock.Lock()
	defer b.liveIDTypeMapLock.Unlock()
	b.liveIDTypeMap[liveID] = reflectType
}

// RegisterLiveIDType registers the type of the liveID instance,
// allowing the actual type of liveID to be obtained when injecting a brick for an interface.
func RegisterLiveIDType[T Brick](liveID string) {
	brickManager.RegisterLiveIDType(liveID, reflect.TypeOf((*(new(T)))))
}

type RegisterBrickParam struct {
	TypeID       string
	ReflectType  reflect.Type
	Lives        []Live
	BrickFactory func(jsonConf []byte) Brick
}

func (b *BrickManager) register2(typeID string, reflectType reflect.Type, brickFactory ...func(jsonConf []byte) Brick) {
	param := RegisterBrickParam{
		TypeID:      typeID,
		ReflectType: reflectType,
	}
	if len(brickFactory) != 0 {
		param.BrickFactory = brickFactory[0]
	}
	b.register(param)
}

// register registers a brick type and its all recursive dependencies.
// It takes a type ID, the type itself and optionally a factory function.
//
// If a factory function is provided, it will be used to create new instances of the brick type from a configuration.
// If not, the brick is registered as a non-configurable brick and a default instance will be used.
func (b *BrickManager) register(param RegisterBrickParam) {
	typeID, reflectType, lives, brickFactory := param.TypeID, param.ReflectType, param.Lives, param.BrickFactory
	if !b.setBrickTypeID(reflectType, typeID) {
		return
	}
	// fmt.Println("RegisterBrickFactory", TypeID, reflectType)
	if brickFactory != nil {
		b.brickFactoriesLock.Lock()
		b.brickFactories[typeID] = func(config any) Brick {
			config = handleConfig(config)
			configBytes, err := json.Marshal(config)
			if err != nil {
				panic(fmt.Errorf("brick config marshal error: %w", err))
			}
			if bytes.Equal(configBytes, []byte("null")) {
				configBytes = nil
			}
			return brickFactory(configBytes)
		}
		b.brickFactoriesLock.Unlock()
	}

	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	if reflectType.Kind() != reflect.Struct {
		return
	}
	var brickFieldNames = make(map[string]bool, 10)
	for i := 0; i < reflectType.NumField(); i++ {
		Field := reflectType.Field(i)
		fieldType := Field.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() != reflect.Struct {
			continue
		}
		tag, ok := Field.Tag.Lookup(brickTag)
		if !ok {
			continue
		}
		brickFieldNames[Field.Name] = true
		liveID, typeID, isClone, _ := b.parseTag(tag)
		if !isClone && liveID != typeID && liveID != "" {
			b.setDeclaredLiveID(liveID)
		}
		// This is the dependency that needs to be injected, check if it implements the Brick interface
		brickType := reflect.TypeOf((*Brick)(nil)).Elem()
		imp := fieldType.Implements(brickType)
		ptrImp := reflect.PointerTo(fieldType).Implements(brickType)
		if !imp && !ptrImp {
			panic(fmt.Errorf("field %s in %s is not a brick component", Field.Name, reflectType))
		}
		// Call BrickTypeID()
		instance := reflect.New(fieldType)
		if !ptrImp {
			instance = instance.Elem()
		}
		instanceI := instance.Interface()
		fieldTypeID := instanceI.(Brick).BrickTypeID()
		instanceConfiger, ok := instanceI.(BrickNewer)
		instanceLives, ok2 := instanceI.(BrickLives)
		registerType := fieldType
		if ptrImp {
			registerType = reflect.PointerTo(fieldType)
		}
		params := RegisterBrickParam{
			TypeID:      fieldTypeID,
			ReflectType: registerType,
		}
		if ok {
			params.BrickFactory = func(jsonConf []byte) Brick {
				return instanceConfiger.NewBrick(jsonConf)
			}
		}
		if ok2 {
			params.Lives = instanceLives.BrickLives()
		}
		b.register(params)
	}
	for _, live := range lives {
		for field := range live.RelyLives {
			if !brickFieldNames[field] {
				panic(fmt.Errorf("field %s in %s is not a brick component", field, reflectType))
			}
		}
	}
}

func (b *BrickManager) parseTag(tag string) (liveID string, typeID string, isClone bool, isRandomLiveID bool) {
	if tag == "random" {
		isRandomLiveID = true
		return
	}
	ids := strings.Split(tag, ",")
	liveID = ids[0]
	typeID = ""
	if len(ids) >= 2 {
		typeID = ids[1]
	}
	if strings.HasPrefix(tag, "clone:") || tag == "clone" {
		isClone = true
		liveID = strings.TrimPrefix(liveID, "clone:")
		if liveID == "clone" {
			liveID = ""
		}
	}
	return
}

func (b *BrickManager) getTypeIDByReflectType(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	newInstancePtr := reflect.New(typ)
	brick, ok := newInstancePtr.Interface().(Brick)
	if ok {
		return brick.BrickTypeID()
	}
	brick, ok = newInstancePtr.Elem().Interface().(Brick)
	if ok {
		return brick.BrickTypeID()
	}
	panic(fmt.Errorf("type %s is not a brick", typ))
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
		if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
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

// GetOrCreate retrieves a brick instance, creating it if necessary.
// If liveID is not provided, it will use the typeID as the LiveID.
//
// Whether non-pointer type singleton components can be shared needs to be ensured by the user.
func GetOrCreate[T Brick](liveID ...string) T {
	brickManager.brickConfigCheckOnce.Do(brickManager.checkConfig)
	return getBrickInstance(reflect.TypeOf((*(new(T)))), true, liveID...).Interface().(T)
}

func Get[T Brick](liveID ...string) T {
	brickManager.brickConfigCheckOnce.Do(brickManager.checkConfig)
	return getBrickInstance(reflect.TypeOf((*(new(T)))), false, liveID...).Interface().(T)
}

// Interface type is not a brick type, but a brick can be injected into an interface type.
func getBrickInstance(brickType reflect.Type, createUnknown bool, liveID ...string) reflect.Value {
	// fmt.Println("getBrickInstance2", brickType)
	typeID, ok := brickManager.getBrickTypeID(brickType)
	if !ok {
		switch brickType.Kind() {
		case reflect.Ptr:
			_, ok = brickManager.getBrickTypeID(brickType.Elem())
			if !ok {
				panic(fmt.Errorf("this brick type is not registered: %s", brickType))
			}
			ptrInstance := reflect.New(brickType.Elem())
			instance := getBrickInstance(brickType.Elem(), createUnknown, liveID...)
			ptrInstance.Elem().Set(instance)
			return ptrInstance
		default:
			typePtr := reflect.PointerTo(brickType)
			_, ok = brickManager.getBrickTypeID(typePtr)
			if !ok {
				panic(fmt.Errorf("this brick type is not registered: %s", brickType))
			}
			ptrInstance := getBrickInstance(typePtr, createUnknown, liveID...)
			return ptrInstance.Elem()
		}
	}
	targetLiveID := ""
	if len(liveID) > 0 && liveID[0] != "" {
		targetLiveID = liveID[0]
	} else {
		targetLiveID = typeID
	}

	if targetLiveID != typeID {
		if _, ok := brickManager.getBrickType(targetLiveID); ok {
			panic(fmt.Errorf("liveID(%s) is not allowed to be the same as the typeID of other brick", targetLiveID))
		}
	}

	brick, ok := brickManager.getBrickFromExist(targetLiveID)
	if ok {
		return convertInstance(brick, brickType)
	}
	if !createUnknown && targetLiveID != typeID && !brickManager.getDeclaredLiveID(targetLiveID) {
		panic(fmt.Sprintf("liveID(%s) is not explicitly declared in the configuration or tag, you can use GetOrCreate to create it", targetLiveID))
	}

	ok = brickManager.getBrickBuilding(targetLiveID)
	if ok {
		panic(fmt.Errorf("circular dependency detected, liveID: %s", targetLiveID))
	}
	brickManager.setBrickBuilding(targetLiveID, true)
	defer brickManager.setBrickBuilding(targetLiveID, false)

	brickConfig, configExist := brickManager.getBrickConfig(targetLiveID)
	if configExist {
		if brickConfig.LiveID != targetLiveID {
			panic(fmt.Errorf("config liveID mismatch: ID(%s) != ID(%s)", brickConfig.LiveID, targetLiveID))
		}
		if brickConfig.TypeID != typeID {
			panic(fmt.Errorf("config TypeID mismatch: ID(%s) != ID(%s)", brickConfig.TypeID, typeID))
		}
	}
	brickParser, parserExist := brickManager.getBrickFactory(typeID)
	if !parserExist {
		ret := createEmptyInstance(brickType)
		if ret.Type().Kind() == reflect.Ptr {
			ret = injectBrick(ret, targetLiveID)
		} else {
			ret = injectBrick(wrapPointerLayer(ret), targetLiveID)
		}
		brickManager.saveBrickInstance(targetLiveID, ret)
		return convertInstance(ret, brickType)
	}
	t := brickParser(brickConfig.Config)
	ret := reflect.ValueOf(t)
	if !isSameBaseType(ret.Type(), brickType) {
		panic(fmt.Errorf("brick(%s) %v NewBrick method return error type: %v", typeID, brickType, ret.Type()))
	}
	//todo: test for type == interface
	if ret.Type().Kind() == reflect.Ptr {
		ret = injectBrick(ret, targetLiveID)
	} else {
		ret = injectBrick(wrapPointerLayer(ret), targetLiveID)
	}
	// fmt.Println("injectBrick ret", ret)
	brickManager.saveBrickInstance(targetLiveID, ret)
	return convertInstance(ret, brickType)
}

func isSameBaseType(typ1 reflect.Type, typ2 reflect.Type) bool {
	for typ1.Kind() == reflect.Ptr {
		typ1 = typ1.Elem()
	}
	for typ2.Kind() == reflect.Ptr {
		typ2 = typ2.Elem()
	}
	return typ1 == typ2
}

func convertInstance(instance reflect.Value, targetType reflect.Type) reflect.Value {
	// fmt.Println("convertInstance", instance.Type(), targetType)
	if instance.Type() == targetType {
		return instance
	}
	retLevel := getPointerLevel(instance.Type())
	if targetType.Kind() == reflect.Interface {
		if retLevel > 1 {
			for i := 0; i < retLevel-1; i++ {
				instance = instance.Elem()
			}
		}
		if retLevel == 0 && !instance.Type().Implements(targetType) {
			instance = wrapPointerLayer(instance)
		}
		return instance
	}
	brickTypeLevel := getPointerLevel(targetType)
	if retLevel > brickTypeLevel {
		for i := 0; i < retLevel-brickTypeLevel; i++ {
			instance = instance.Elem()
		}
		return instance
	}
	if retLevel < brickTypeLevel {
		for i := 0; i < brickTypeLevel-retLevel; i++ {
			instance = wrapPointerLayer(instance)
		}
	}
	return instance
}

// Get the pointer level of the type
func getPointerLevel(typ reflect.Type) int {
	level := 0
	for typ.Kind() == reflect.Ptr {
		level++
		typ = typ.Elem()
	}
	return level
}

// Wrap a reflect.Value with a pointer layer
func wrapPointerLayer(value reflect.Value) reflect.Value {
	var ret = reflect.New(value.Type())
	ret.Elem().Set(value)
	return ret
}

func createEmptyInstance(typ reflect.Type) reflect.Value {
	if typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Interface {
		instance := createEmptyInstance(typ.Elem())
		ptrInstance := reflect.New(typ.Elem())
		ptrInstance.Elem().Set(instance)
		return ptrInstance
	}
	return reflect.New(typ).Elem()
}

// injectBrick injects dependencies into a brick instance by looking for fields with the `brick` tag.
func injectBrick(brick reflect.Value, brickLiveID string) reflect.Value {
	// fmt.Println("injectBrick", brick)
	rfValue := brick
	for rfValue.Kind() == reflect.Ptr || rfValue.Kind() == reflect.Interface {
		rfValue = rfValue.Elem()
	}
	if rfValue.Kind() != reflect.Struct {
		return brick
	}
	rfType := rfValue.Type()
	var brickLive *Live
	lives, ok := getBrickLives(rfType)
	if ok {
		for _, live := range lives {
			if live.LiveID == brickLiveID {
				brickLive = &live
				break
			}
		}
	}
	for i := 0; i < rfType.NumField(); i++ {
		typeField := rfType.Field(i)
		valueField := rfValue.Field(i)
		if !valueField.CanSet() {
			continue
		}
		if tag, ok := typeField.Tag.Lookup(brickTag); ok {
			typ := valueField.Type()
			if brickLive != nil {
				if tag2, ok := brickLive.RelyLives[typeField.Name]; ok {
					tag = tag2
				}
			}
			if typ.Kind() == reflect.Interface {
				injectInterfaceBrick(valueField, tag)
				continue
			}
			liveID, _, isClone, isRandomLiveID := brickManager.parseTag(tag)
			if isRandomLiveID {
				liveID = RandomLiveID()
			}
			if isClone {
				if liveID == "" {
					liveID, ok = brickManager.getBrickTypeID(typ)
					if !ok {
						panic(fmt.Errorf("unexpect error, brick type(%s) not found", typ))
					}
				}
				valueField.Set(cloneBrick2(typ, liveID))
			} else {
				valueField.Set(getBrickInstance(typ, true, liveID))
			}
		}
	}
	return brick
}

// `brick:"liveID,typeID"`
func injectInterfaceBrick(valueField reflect.Value, tag string) {
	liveID, typeID, cloneBrick, isRandomLiveID := brickManager.parseTag(tag)
	if isRandomLiveID {
		panic(fmt.Errorf("interface type brick(%s) cannot use random liveID", valueField.Type()))
	}
	if liveID == "" {
		if typeID == "" {
			panic(fmt.Errorf("interface type brick(%s) must give a liveID on tag", valueField.Type()))
		}
		liveID = typeID
	}
	brick, ok := brickManager.getBrickFromExist(liveID)
	if ok {
		if cloneBrick {
			valueField.Set(cloneBrick2(brick.Type(), liveID))
		} else {
			// fmt.Println("convertInstance", brick.Type(), valueField.Type())
			valueField.Set(convertInstance(brick, valueField.Type()))
		}
		return
	}
	brickconf, ok := brickManager.getBrickConfig(liveID)
	if ok {
		if typeID != "" && brickconf.TypeID != typeID {
			panic(fmt.Errorf("the interface brick(%v) TypeID mismatch: config(%s) != tag(%s)", valueField.Type(), brickconf.TypeID, typeID))
		}
		typ, ok := brickManager.getBrickType(brickconf.TypeID)
		if !ok {
			panic(fmt.Errorf("the interface brick(%v) dependency not found, typeID(%s)", valueField.Type(), brickconf.TypeID))
		}
		if cloneBrick {
			valueField.Set(cloneBrick2(typ, liveID))
		} else {
			valueField.Set(getBrickInstance(typ, true, liveID))
		}
		return
	}
	if typeID != "" {
		typ, ok := brickManager.getBrickType(typeID)
		if !ok {
			panic(fmt.Errorf("the interface brick(%v) dependency not found, typeID(%s)", valueField.Type(), typeID))
		}
		if cloneBrick {
			valueField.Set(cloneBrick2(typ, liveID))
		} else {
			valueField.Set(getBrickInstance(typ, true, liveID))
		}
		return
	}
	// Get the type of the liveID that the user has registered
	brickManager.liveIDTypeMapLock.RLock()
	typ, ok := brickManager.liveIDTypeMap[liveID]
	brickManager.liveIDTypeMapLock.RUnlock()
	// if !ok {
	// 	// Try to use liveID as typeID, get the default instance of the type.
	// 	// We decided not to do this, because it adds to the user's cognitive burden,
	// 	// and to achieve the same effect,
	// 	// you can clearly use `brick:",typeID"` to represent the default instance of the type
	// 	typ, ok = brickManager.getBrickType(liveID)
	// }
	if ok {
		if cloneBrick {
			valueField.Set(cloneBrick2(typ, liveID))
		} else {
			valueField.Set(getBrickInstance(typ, true, liveID))
		}
		return
	}
	panic(fmt.Errorf("the interface brick(%v) dependency not found, can't determine the type of liveID(%s)", valueField.Type(), liveID))
}

func CloneConfig[T Brick](liveID ...string) (newLiveID string) {
	cloneId := ""
	if len(liveID) > 0 && liveID[0] != "" {
		cloneId = liveID[0]
	} else {
		cloneId = brickManager.getTypeIDByReflectType(reflect.TypeOf((*(new(T)))))
	}
	newLiveID = RandomLiveID()
	brickConfig, ok := brickManager.getBrickConfig(cloneId)
	if !ok {
		panic(fmt.Errorf("liveID(%s) does not have a configuration", cloneId))
	}
	brickManager.setBrickConfig(newLiveID, brickConfig)
	brickManager.setDeclaredLiveID(newLiveID)
	return newLiveID
}

func cloneBrick(brickType reflect.Type, liveID string) (newBrick reflect.Value, newLiveID string) {
	newLiveID = RandomLiveID()
	brickConfig, ok := brickManager.getBrickConfig(liveID)
	if ok {
		brickManager.setBrickConfig(newLiveID, brickConfig)
	}
	return getBrickInstance(brickType, true, newLiveID), newLiveID
}

func cloneBrick2(brickType reflect.Type, liveID string) reflect.Value {
	b, _ := cloneBrick(brickType, liveID)
	return b
}

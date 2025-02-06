package brick

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

var (
	brickManager = &BrickManager{
		brickConfigs:     make(map[string]BrickConfig),
		instances:        make(map[string]reflect.Value),
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

	// buildingBrickGroup is a group of bricks that are being built, indexed by LiveID.
	buildingBrickGroup singleflight.Group

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

func isSameBaseType(typ1 reflect.Type, typ2 reflect.Type) bool {
	for typ1.Kind() == reflect.Ptr {
		typ1 = typ1.Elem()
	}
	for typ2.Kind() == reflect.Ptr {
		typ2 = typ2.Elem()
	}
	return typ1 == typ2
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

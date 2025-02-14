package brick

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

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

func GetBrickTypeID[T Brick]() string {
	var instance = *new(T)
	var typ = reflect.TypeOf(instance)
	if typ.Kind() != reflect.Ptr {
		return instance.BrickTypeID()
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	newInstancePtr := reflect.New(typ)
	return newInstancePtr.Interface().(Brick).BrickTypeID()
}

// RegisterNewer like Register, but it also registers a factory for configuration parsing.
// The provided type must implement the BrickNewer interface, which includes the NewBrick method
// for parsing configurations.
func RegisterNewer[T BrickNewer]() {
	// Pointer receiver registers pointer type, value receiver registers value type
	var instance = *new(T)
	var typ = reflect.TypeOf(instance)
	var check = func() {
		baseField, ok := typ.FieldByName("BrickBase")
		if ok && baseField.Anonymous && baseField.Type.Kind() == reflect.Ptr {
			panic(fmt.Errorf("brick base field of %s can't be a pointer", typ))
		}
	}
	if typ.Kind() != reflect.Ptr {
		check()
		brickManager.register2(instance.BrickTypeID(), typ, instance.NewBrick)
		return
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	check()
	newInstancePtr := reflect.New(typ)
	if typ.Implements(brickNewerInterfaceType) {
		brick, _ := newInstancePtr.Elem().Interface().(BrickNewer)
		brickManager.register2(brick.BrickTypeID(), typ, brick.NewBrick)
		return
	}
	brick, _ := newInstancePtr.Interface().(BrickNewer)
	brickManager.register2(brick.BrickTypeID(), newInstancePtr.Type(), brick.NewBrick)
}

// RegisterLives like RegisterNewer, but it requires the brick to implement the BrickLives interface,
// which describes the specified instance's specified dependency relationship.
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
	var check = func() {
		baseField, ok := typ.FieldByName("BrickBase")
		if ok && baseField.Anonymous && baseField.Type.Kind() == reflect.Ptr {
			panic(fmt.Errorf("brick base field of %s can't be a pointer", typ))
		}
	}
	if typ.Kind() != reflect.Ptr {
		check()
		param.ReflectType = typ
		param.TypeID = instance.BrickTypeID()
		param.Lives = instance.BrickLives()
		param.BrickFactory = instance.NewBrick
		return
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	check()
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
			params.BrickFactory = instanceConfiger.NewBrick
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

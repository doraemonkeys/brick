package brick

import (
	"fmt"
	"reflect"
	"unsafe"
)

// GetOrCreate like Get, but it will create a new instance for unknown liveID.
func GetOrCreate[T Brick](liveID ...string) T {
	brickManager.brickConfigCheckOnce.Do(brickManager.checkConfig)
	ctx := getBrickInstanceCtx{
		buildingBrick: make(map[reflect.Type]bool),
		createUnknown: true,
	}
	return getBrickInstance(reflect.TypeOf((*(new(T)))), ctx, liveID...).Interface().(T)
}

// Get retrieves a brick instance, creating it if necessary.
// If liveID is not provided, it will use the typeID as the LiveID.
//
// If an instance for the given liveID has never been created before, and this is an unknown liveID,
// the function will panic to prevent unexpected behavior.
// An unknown liveID is one that has not been declared in the configuration or tag,
// and has not been explicitly registered as a non-default liveID. The default instance's liveID is the same as the typeID.
//
// When retrieving a non-pointer Brick instance, be aware of whether the type can be copied safely. No checks are performed for this.
func Get[T Brick](liveID ...string) T {
	brickManager.brickConfigCheckOnce.Do(brickManager.checkConfig)
	ctx := getBrickInstanceCtx{

		buildingBrick: make(map[reflect.Type]bool),
		createUnknown: false,
	}
	return getBrickInstance(reflect.TypeOf((*(new(T)))), ctx, liveID...).Interface().(T)
}

type getBrickInstanceCtx struct {
	// Don't save the type of the dereferenced pointer, because if there is a circular dependency, it will save the same type twice, causing a panic.
	buildingBrick map[reflect.Type]bool
	createUnknown bool
}

// Interface type is not a brick type, but a brick can be injected into an interface type.
//
// The instance type obtained from the same liveID may be a struct, or a *struct, depending on the type of brickType.
func getBrickInstance(brickType reflect.Type, ctx getBrickInstanceCtx, liveID ...string) reflect.Value {
	// fmt.Println("getBrickInstance2", brickType)
	typeID, ok := brickManager.getBrickTypeID(brickType)

	if !ok {
		switch brickType.Kind() {
		case reflect.Ptr:
			wrappedBrickType := brickType
			for wrappedBrickType.Kind() == reflect.Ptr {
				typeID, ok = brickManager.getBrickTypeID(wrappedBrickType.Elem())
				if ok {
					break
				}
				wrappedBrickType = wrappedBrickType.Elem()
			}
			if !ok {
				panic(fmt.Errorf("this brick type is not registered: %s", brickType))
			}
		default:
			typePtr := reflect.PointerTo(brickType)
			_, ok = brickManager.getBrickTypeID(typePtr)
			if !ok {
				panic(fmt.Errorf("this brick type is not registered: %s", brickType))
			}
			ptrInstance := getBrickInstance(typePtr, ctx, liveID...)
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
	if !ctx.createUnknown && targetLiveID != typeID && !brickManager.getDeclaredLiveID(targetLiveID) {
		panic(fmt.Sprintf("liveID(%s) is not explicitly declared in the configuration or tag, you can use GetOrCreate to create it", targetLiveID))
	}

	if ctx.buildingBrick[brickType] {
		panic(fmt.Errorf("circular dependency detected, brickType: %v, liveID: %s", brickType, targetLiveID))
	}
	ctx.buildingBrick[brickType] = true
	defer func() {
		ctx.buildingBrick[brickType] = false
	}()

	v, _, _ := brickManager.buildingBrickGroup.Do(targetLiveID, func() (any, error) {
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
			ret := createEmptyPtrInstance(brickType)
			ret = injectBrick(ret, targetLiveID, ctx)
			brickManager.saveBrickInstance(targetLiveID, ret)
			return convertInstance(ret, brickType), nil
		}
		t := brickParser(brickConfig.Config)
		ret := reflect.ValueOf(t)

		if !isSameBaseType(ret.Type(), brickType) {
			panic(fmt.Errorf("brick(%s) %v NewBrick method return error type: %v", typeID, brickType, ret.Type()))
		}
		//todo: test for interface type
		if ret.Type().Kind() == reflect.Ptr {
			ret = injectBrick(ret, targetLiveID, ctx)
		} else {
			ret = injectBrick(wrapPointerLayer(ret), targetLiveID, ctx)
		}

		// fmt.Println("injectBrick ret", ret)
		brickManager.saveBrickInstance(targetLiveID, ret)
		return convertInstance(ret, brickType), nil
	})
	return v.(reflect.Value)
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

// injectBrick injects dependencies into a brick instance by looking for fields with the `brick` tag.
func injectBrick(brick reflect.Value, brickLiveID string, ctx getBrickInstanceCtx) reflect.Value {
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
		if typeField.Anonymous && typeField.Name == "BrickBase" {
			if typeField.Type.Kind() == reflect.Ptr {
				valueField.Set(reflect.New(typeField.Type.Elem()))
				// Get the unexported field.
				unexportedField := valueField.Elem().FieldByName("liveID")
				// Use unsafe to get an addressable value.
				unsafeField := reflect.NewAt(unexportedField.Type(), unsafe.Pointer(unexportedField.UnsafeAddr())).Elem()
				unsafeField.SetString(brickLiveID)
			} else {
				// valueField.FieldByName("liveID").SetString(brickLiveID)
				unexportedField := valueField.FieldByName("liveID")
				unsafeField := reflect.NewAt(unexportedField.Type(), unsafe.Pointer(unexportedField.UnsafeAddr())).Elem()
				unsafeField.SetString(brickLiveID)
			}
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
				injectInterfaceBrick(valueField, tag, ctx)
				continue
			}
			var newCtx = ctx
			liveID, _, isClone, isRandomLiveID := brickManager.parseTag(tag)
			if isRandomLiveID {
				liveID = RandomLiveID()
				newCtx.createUnknown = true
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
				valueField.Set(getBrickInstance(typ, newCtx, liveID))
			}
		}
	}

	return brick
}

// `brick:"liveID,typeID"`
func injectInterfaceBrick(valueField reflect.Value, tag string, ctx getBrickInstanceCtx) {
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
			valueField.Set(getBrickInstance(typ, ctx, liveID))
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
			valueField.Set(getBrickInstance(typ, ctx, liveID))
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
			valueField.Set(getBrickInstance(typ, ctx, liveID))
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
	ctx := getBrickInstanceCtx{
		buildingBrick: make(map[reflect.Type]bool),
		createUnknown: true,
	}
	return getBrickInstance(brickType, ctx, newLiveID), newLiveID
}

func cloneBrick2(brickType reflect.Type, liveID string) reflect.Value {
	b, _ := cloneBrick(brickType, liveID)
	return b
}

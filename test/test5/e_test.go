package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/doraemonkeys/brick"
)

type Mover interface {
	Move()
}

// ExampleBrick is a simple brick implementation.
type ExampleBrick struct {
	ID    string
	Name  string
	Dep   *DependencyBrick  `brick:"dep"`
	Dep2  *DependencyBrick  `brick:"dep2"`
	Dep22 **DependencyBrick `brick:"dep2"`
	Dep3  DependencyBrick   `brick:"clone:dep2"`
	Dep4  Mover             `brick:"clonedep2,DogTypeID"`
	Dep5  Mover             `brick:"dep2"`
	Dep6  Mover             `brick:"clone:dep2"`
}

const ExampleBrickTypeID = "ExampleBrick"

func (e *ExampleBrick) BrickTypeID() string {
	return ExampleBrickTypeID
}

func (e *ExampleBrick) NewBrick(jsonConf []byte) brick.Brick {
	cfg := &ExampleBrickConfig{}
	json.Unmarshal(jsonConf, cfg)
	var newE = &ExampleBrick{}
	newE.Name = cfg.Name
	return newE
}

// DependencyBrick is another brick that ExampleBrick depends on.
type DependencyBrick struct {
	Value string
	// Dep   *DependencyBrick `brick:"dep3"`
	Dep2 *Dog2 `brick:"depDog2"`
	Conf *DependencyBrickConfig
}

const DependencyBrickTypeID = "DependencyBrick"

func (d *DependencyBrick) BrickTypeID() string {
	return DependencyBrickTypeID
}

func (d *DependencyBrick) NewBrick(jsonConf []byte) brick.Brick {
	fmt.Println("DependencyBrick ParseBrickConfig jsonConf", string(jsonConf))
	cfg := &DependencyBrickConfig{}
	json.Unmarshal(jsonConf, cfg)
	newD := &DependencyBrick{}
	newD.Value = cfg.Value
	return newD
}

func (d *DependencyBrick) Move() {
	fmt.Println("DependencyBrick Move")
}

// Config holds the configuration for ExampleBrick
type ExampleBrickConfig struct {
	Name string
}

// Config holds the configuration for DependencyBrick
type DependencyBrickConfig struct {
	Value string
}

type Dog struct{}

func (d *Dog) Move() {
	fmt.Println("Dog is moving", d)
}

func (d *Dog) BrickTypeID() string {
	return "DogTypeID"
}

type Dog2 struct{}

func (d *Dog2) Move() {
	fmt.Println("Dog2 is moving", d)
}

func (d *Dog2) BrickTypeID() string {
	return "Dog2TypeID"
}

func init() {
	// Register the brick parsers
	// brick.RegisterBrickFactory(ExampleBrickTypeID, func(config []byte) *ExampleBrick {
	// 	cfg := &ExampleBrickConfig{}
	// 	json.Unmarshal(config, cfg)
	// 	return &ExampleBrick{
	// 		ID:   "example-id",
	// 		Name: cfg.Name,
	// 	}
	// })

	// brick.Register[*ExampleBrick]()
	// brick.Register[*DependencyBrick]()
	// brick.RegisterDefaultConfig[*ExampleBrick]()
	// brick.RegisterWithConfig(func(brick **ExampleBrick, config *ExampleBrickConfig) { (*brick).Name = config.Name })
	brick.RegisterNewer[*ExampleBrick]()
	brick.Register[*Dog]()
	// brick.Register[*DependencyBrick]()

	// brick.RegisterBrickFactory(DependencyBrickTypeID, func(config []byte) DependencyBrick {
	// 	cfg := &DependencyBrickConfig{}
	// 	json.Unmarshal(config, cfg)
	// 	return DependencyBrick{
	// 		Value: cfg.Value,
	// 	}
	// })
	// brick.RegisterBrickFactory(DependencyBrickTypeID, brick.DefaultBrickFactory[*DependencyBrick])
}

func Test_2(t *testing.T) {
	// Create a temporary config file
	configFile := `
{
    "Bricks": [
		{
            "MetaData": {
                "Name": "ExampleBrickConfig",
                "TypeID": "ExampleBrick"
            },
            "Lives": [
                {
                    "LiveID": "ExampleBrick",
                    "Config": {
                        "Name": "My Example Brick"
                    }
                }
            ]
        },
		{
			"MetaData": {
				"Name": "DependencyBrickConfig",
                "TypeID": "DependencyBrick"
            },
            "Lives": [
                {
                    "LiveID": "dep",
                    "Config": {
                        "Value": "${http_proxy}"
                    }
                },
				{
					"LiveID": "dep2",
					"Config": {
						"Value": "hhhhh哈哈哈哈"
					}
				},
				{
					"LiveID": "DependencyBrick",
					"Config": {
						"Value": "dep3"
					}
				}
            ]
        }
    ]
}
`

	tmpFile, err := os.CreateTemp("", "brick-config-*.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configFile)
	if err != nil {
		panic(err)
	}
	tmpFile.Close()

	// Load configurations
	err = brick.AddConfigFile(tmpFile.Name())
	if err != nil {
		panic(err)
	}

	// Get the brick instance using LiveID
	exampleBrick := brick.GetOrCreate[*ExampleBrick]()
	fmt.Println("Example Brick ID:", exampleBrick.ID)
	fmt.Println("Example Brick Name:", exampleBrick.Name)
	fmt.Println("Example Brick Dep:", exampleBrick.Dep.Value)
	fmt.Println("Example Brick Dep2:", exampleBrick.Dep2.Value)
	fmt.Println("Example Brick Dep22:", (*exampleBrick.Dep22).Value)
	fmt.Println("Example Brick Dep3:", exampleBrick.Dep3.Value)
	fmt.Println("Example Brick Dep4:", exampleBrick.Dep4)
	fmt.Println("Example Brick Dep5:", exampleBrick.Dep5)
	fmt.Println("Example Brick Dep6:", exampleBrick.Dep6)
	exampleBrick.Dep4.Move()
	exampleBrick.Dep5.Move()
	exampleBrick.Dep6.Move()
	exampleBrick.Dep.Dep2.Move()

	// Get the brick instance using TypeID, it will default to the typeID as liveID
	// dependencyBrick := brick.GetBrick[*DependencyBrick]("DependencyBrick444")
	// fmt.Println("Dependency Brick Value:", dependencyBrick)

	//clone
	cloneBrick := brick.Get[*ExampleBrick](brick.CloneConfig[*ExampleBrick]())
	fmt.Println("Clone Brick ID:", cloneBrick.ID)
	fmt.Println("Clone Brick Name:", cloneBrick.Name)
	fmt.Println("Clone Brick Dep:", cloneBrick.Dep.Value)
	fmt.Println("Clone Brick Dep2:", cloneBrick.Dep2.Value)

}

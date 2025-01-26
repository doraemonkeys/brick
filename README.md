<h3 align="center"> English | <a href='README-ZH.md'>中文</a></h3>


# Brick

Brick is a lightweight and non-intrusive library designed for component-based configuration management and dependency injection. "Lightweight" means the library has a simple implementation and is easy to use. "Non-intrusive" means that Brick is not a framework but rather a library; you can easily leverage its dependency injection and configuration management features without modifying the logic of your existing code.



Unlike libraries that strive to maximize support for default behaviors, Brick prioritizes certainty for developers. It thoroughly checks for potential injection errors and throws a `panic` during startup to ensure they are addressed immediately. This allows you to use Brick confidently, without constantly needing to worry about hidden issues.

## Getting Started

### Creating a Brick Component

Any type that implements the `Brick` interface is considered a Brick component.  You only need to implement the `BrickTypeID` method, which should return a globally unique constant string that serves as the type ID of the component.

```go
type Brick1 struct {}

func (l *Brick1) BrickTypeID() string {
    return "EG7LGZ4SX2O9L7ZZHS64"
}
```

### Injecting Other Brick Components

To inject other Brick components, simply add the `brick` tag to the field representing the dependency. Brick will automatically handle the injection.

```go
type Brick2 struct {
    Brick1 *Brick1 `brick:""`
}

func (l *Brick2) BrickTypeID() string {
    return "QQZWZH166CJIKAAAFRHV"
}
```

Let's break down the meaning of this tag: `brick` indicates that the Brick library should automatically inject a dependency. The string after `brick:` specifies the *liveID* of the component instance to be injected. `""` (an empty string) means that the *default liveID* (the default instance) for this component should be injected, which in this case is the `TypeID` of `Brick1`.



This is equivalent to:

```go
type Brick2 struct {
    Brick1 *Brick1 `brick:"EG7LGZ4SX2O9L7ZZHS64"`
}
```

Of course, you can also specify a different globally unique `liveID`.

### Parsing Configuration for Brick Instances

If a Brick component needs to read external configuration from a file, it must implement the `BrickNewer` interface. The `NewBrick` method of this interface must parse the incoming JSON string representing the configuration and return a new instance of the Brick component.

```go
import (
    "encoding/json"
    "github.com/your-repo/brick" // Assuming this is your import path
)


type Brick2 struct {
    ConfigString string `json:"configString" yaml:"configString" toml:"configString"`
    Brick1 *Brick1 `brick:""`
}

func (l *Brick2) NewBrick(jsonConfig []byte) brick.Brick {
    var newBrick = &Brick2{}
    _ = json.Unmarshal(jsonConfig, newBrick)
    return newBrick
}
```

As you can see from the field tags, configuration files can use popular formats like JSON, YAML, and TOML.



Example configuration:

```json
[
    {
        "metaData": {
            "name": "Brick2 Instances",
            "typeID": "QQZWZH166CJIKAAAFRHV"
        },
        "lives": [
            {
                "liveID": "QQZWZH166CJIKAAAFRHV",
                "config": {
                    "configString": "your config string"
                }
            }
        ]
    }
]
```

You also need to inform the Brick library about the location of your configuration file, like this:

```go
err := brick.AddConfigFile("config.json")
```

### Registering Brick Components

You need to call the `Register` method to register your top-level Brick components.  Dependencies will automatically be scanned and registered.

```go
brick.RegisterNewer[*Brick2]()
```

### Getting Brick Component Instances

You can retrieve an instance of a Brick component using the `Get` or `GetOrCreate` methods:

```go
brick1 := brick.Get[*Brick1]()
```

### Example Code

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/doraemonkeys/brick"
)

type Brick1 struct{}

func (l *Brick1) BrickTypeID() string {
	return "EG7LGZ4SX2O9L7ZZHS64"
}

func (l *Brick1) SayHello() string {
	return "Hello, Brick1"
}

type Brick2 struct {
	ConfigString string  `json:"configString" yaml:"configString" toml:"configString"`
	Brick1       *Brick1 `brick:""`
}

func (l *Brick2) BrickTypeID() string {
	return "QQZWZH166CJIKAAAFRHV"
}

func (l *Brick2) NewBrick(jsonConfig []byte) brick.Brick {
	var newBrick = &Brick2{}
	if len(jsonConfig) > 0 {
		if err := json.Unmarshal(jsonConfig, newBrick); err != nil {
			panic(fmt.Errorf("Brick2 config error: %w", err))
		}
	}
	return newBrick
}

func (l *Brick2) SayHello() {
	fmt.Println("Hello, Brick2")
	fmt.Println(l.Brick1.SayHello())
	fmt.Printf("Brick2 config: %v", l.ConfigString)
}

const JsonConfig = `
[
    {
        "metaData": {
            "name": "Brick2 Instances",
            "typeID": "QQZWZH166CJIKAAAFRHV"
        },
        "lives": [
            {
                "liveID": "QQZWZH166CJIKAAAFRHV",
                "config": {
                    "configString": "your config string"
                }
            }
        ]
    }
]
`

func init() {
	brick.RegisterNewer[*Brick2]()

	_ = os.WriteFile("config.json", []byte(JsonConfig), 0644)
	err := brick.AddConfigFile("config.json")
	if err != nil {
		panic(err)
	}

}

func main() {
	brick2 := brick.Get[*Brick2]()
	brick2.SayHello()
}
```



### Thanks

This library draws inspiration from [dot](https://github.com/scryinfo/dot).


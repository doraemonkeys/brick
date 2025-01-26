<h3 align="center"> 中文 | <a href='README.md'>English</a></h3>



# brick



brick是一个轻量级非侵入式的组件化的配置管理与依赖注入库。轻量意味着Brick库实现逻辑简单使用方便，非侵入式意味着Brick并非是一个框架而是一个库，你可以不修改现有代码的逻辑而轻松使用Brick提供的依赖注入与配置管理功能。



brick库并非最大限度的支持默认行为，相反，为了弥补使用反射给程序员带来的不确定性，它尽可能的对一切异常注入行为进行检查，并在启动时抛出panic以便错误得到及时修正。所以你可以安心使用brick库，不必总是小心翼翼。



## 快速开始

### 创建一个brick组件

任意一个实现了 `Brick` 接口的类型都被视为一个brick组件，你只需要实现  `BrickTypeID` 方法，该方法返回一个全局唯一的常量字符串作为brick组件的类型ID。

```go
type Brick1 struct {}

func (l *Brick1) BrickTypeID() string {
    return "EG7LGZ4SX2O9L7ZZHS64"
}
```

### 在brick组件中注入其他brick组件

你只需要在依赖的brick组件的tag中添加 `brick` 标签，brick库会自动帮你注入。

```go
type Brick2 struct {
    Brick1 *Brick1 `brick:""`
}

func (l *Brick2) BrickTypeID() string {
    return "QQZWZH166CJIKAAAFRHV"
}
```

让我们来解释一下这个tag的含义，`brick` 表示brick库会自动帮你注入这个依赖，后面的字符串表示即将注入组件的实例ID(`liveID`)，`""`空字符串表示这里会注入此组件的**默认liveID(默认实例)**，即 Brick1 的 TypeID。



也就是等价于以下代码：

```go
type Brick2 struct {
    Brick1 *Brick1 `brick:"EG7LGZ4SX2O9L7ZZHS64"`
}
```

当然你也完全可以指定其他全局唯一的liveID。

### 解析 brick 实例的配置文件

如果你依赖的brick组件需要读取外部配置文件，你只需要实现 `BrickNewer` 接口，此接口的 `NewBrick` 方法需要解析传入的配置文件的json字符串，然后返回一个新的 brick 实例即可。

```go
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

通过配置字段的tag可以看出，配置文件支持 json、yaml、toml 这些主流格式。



配置文件示例:

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

当然你必须得告诉brick库你的配置文件在哪里，例如：

```go
err := brick.AddConfigFile("config.json")
```

### 注册brick组件

你需要调用 `Register` 方法注册你顶部的brick组件，其他依赖的brick组件会自动扫描注册。

```go
brick.RegisterNewer[*Brick2]()
```


### 获取brick组件实例

你可以通过调用 `Get` 或 `GetOrCreate` 方法获取brick组件实例。

```go
brick1 := brick.Get[*Brick1]()
```


### 示例代码

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


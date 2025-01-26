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
	fmt.Printf("Brick2 config:%v", l.ConfigString)
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

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/doraemonkeys/brick"
)

// 定义日志组件
type Logger struct {
	Prefix      string `json:"prefix"`
	LogLevel    string `json:"logLevel"`
	EnableColor bool   `json:"enableColor"`
}

func (l *Logger) BrickTypeID() string {
	return "Logger-XU8WQNK1F65QVH4OQ5XY"
}

func (l *Logger) NewBrick(jsonConf []byte) brick.Brick {
	var newLogger = &Logger{}
	err := json.Unmarshal(jsonConf, newLogger)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal logger config: %w", err))
	}
	return newLogger
}

func (l *Logger) BrickTypeLives() []string {
	return []string{"Logger-XU8WQNK1F65QVH4OQ5XY", "Logger-QAAHX1JWAVS4T1KKAN6B"}
}

// 定义数据库连接组件
type DBConnection struct {
	Driver   string  `json:"driver"`
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	DBName   string  `json:"dbName"`
	Logger   *Logger `brick:"Logger-QAAHX1JWAVS4T1KKAN6B"` // 依赖 Logger 组件，注入 liveID 为 "Logger-QAAHX1JWAVS4T1KKAN6B" 的实例
}

func (db *DBConnection) BrickTypeID() string {
	return "DBConnection-742S959CDPWBCD8462A8"
}

func (db *DBConnection) NewBrick(jsonConf []byte) brick.Brick {
	var newDBConn = &DBConnection{}
	err := json.Unmarshal(jsonConf, newDBConn)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal db connection config: %w", err))
	}
	return newDBConn
}

func (db *DBConnection) Connect() {
	db.Logger.Log("Connecting to database...")
	// 模拟数据库连接
	fmt.Printf("Connected to %s database: %s\n", db.Driver, db.DBName)
}

func (l *Logger) Log(message string) {
	if l.EnableColor {
		fmt.Printf("\033[32m[%s] [%s] %s\033[0m\n", l.Prefix, l.LogLevel, message)
	} else {
		fmt.Printf("[%s] [%s] %s\n", l.Prefix, l.LogLevel, message)
	}
}

// 定义用户服务组件
type UserService struct {
	DB       *DBConnection `brick:"DBConnection-1"` // 依赖 DBConnection 组件，注入 liveID 为 "DBConnection-1" 的实例
	MyLogger *Logger       `brick:""`               // 依赖 Logger 组件, 注入 liveID 为 Logger 组件的 TypeID 的实例, 即 "Logger-XU8WQNK1F65QVH4OQ5XY"
}

func (us *UserService) BrickTypeID() string {
	return "UserService-961S959CDPWBCD8462A8"
}

// 定义订单服务组件
type OrderService struct {
	DB *DBConnection `brick:"DBConnection-1"` // 依赖 DBConnection 组件, 注入 liveID 为 "DBConnection-1" 的实例
}

func (os *OrderService) BrickTypeID() string {
	return "OrderService-123S959CDPWBCD8462A8"
}

// 定义一个接口，并注入不同的实例
type IConfig interface {
	GetName() string
}

type AppConfig struct {
	Name string `json:"name"`
}

func (c *AppConfig) BrickTypeID() string {
	return "AppConfig-ABCDEFG"
}

func (c *AppConfig) GetName() string {
	return c.Name
}

func (c *AppConfig) NewBrick(jsonConf []byte) brick.Brick {
	var newAppConfig = &AppConfig{}
	err := json.Unmarshal(jsonConf, newAppConfig)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal app config: %w", err))
	}
	return newAppConfig
}

type DBConfig struct {
	Name string `json:"name"`
}

func (c *DBConfig) BrickTypeID() string {
	return "DBConfig-HIJKLMN"
}

func (c *DBConfig) GetName() string {
	return c.Name
}

func (c *DBConfig) NewBrick(jsonConf []byte) brick.Brick {
	var newDBConfig = &DBConfig{}
	err := json.Unmarshal(jsonConf, newDBConfig)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal db config: %w", err))
	}
	return newDBConfig
}

// 使用接口
type ConfigService struct {
	AppConfig IConfig `brick:"AppConfig-1"` // 注入 liveID 为 "AppConfig-1" 的 AppConfig 实例
	DBConfig  IConfig `brick:"DBConfig-1"`  // 注入 liveID 为 "DBConfig-1" 的 DBConfig 实例
}

func (c *ConfigService) BrickTypeID() string {
	return "ConfigService-OPQRSTU"
}

// 使用克隆实例的组件
type CloneService struct {
	Logger        *Logger        `brick:"clone:Logger-QAAHX1JWAVS4T1KKAN6B"` // 克隆 liveID 为 "Logger-QAAHX1JWAVS4T1KKAN6B" 的 Logger 实例
	Logger2       *Logger        `brick:"clone:"`                            // 克隆 liveID 为 Logger 组件的 TypeID 的实例
	ConfigService *ConfigService `brick:"clone:"`
}

func (c *CloneService) BrickTypeID() string {
	return "CloneService-VWXYZ"
}

// 测试非指针类型的单例
type NonPtrSingleton struct {
	Counter int
	Logger  *Logger `brick:""`
}

func (n *NonPtrSingleton) BrickTypeID() string {
	return "NonPtrSingleton-123456789"
}

func main() {
	// 注册组件
	// brick.RegisterNewer[*Logger]()
	// brick.RegisterNewer[*DBConnection]()
	brick.Register[*UserService]()
	brick.Register[*OrderService]()
	brick.RegisterNewer[*AppConfig]()
	brick.RegisterNewer[*DBConfig]()
	brick.Register[*ConfigService]()
	brick.Register[*CloneService]()
	brick.Register[*NonPtrSingleton]()

	// 注册接口类型对应的实例类型
	brick.RegisterLiveIDType[*AppConfig]("AppConfig-1")
	brick.RegisterLiveIDType[*DBConfig]("DBConfig-1")

	// 添加配置文件
	configFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(`
{
    "bricks": [
        {
            "metaData": {
                "name": "Logger Brick",
                "TypeID": "Logger-XU8WQNK1F65QVH4OQ5XY"
            },
            "lives": [
                {
                    "liveID": "Logger-XU8WQNK1F65QVH4OQ5XY",
                    "config": {
                        "prefix": "My App",
                        "logLevel": "DEBUG",
						"enableColor": true
                    }
                },
                {
                    "liveID": "Logger-QAAHX1JWAVS4T1KKAN6B",
                    "config": {
                        "prefix": "My App2",
                        "logLevel": "INFO",
						"enableColor": false
                    }
                }
            ]
        },
        {
            "metaData": {
                "name": "DBConnection Brick",
                "TypeID": "DBConnection-742S959CDPWBCD8462A8"
            },
            "lives": [
                {
                    "liveID": "DBConnection-1",
                    "config": {
                        "driver": "mysql",
                        "host": "localhost",
                        "port": 3306,
                        "username": "${DB_USER:root}",
                        "password": "${DB_PASSWORD:password}",
                        "dbName": "test"
                    }
                }
            ]
        },
        {
            "metaData": {
                "name": "AppConfig Brick",
                "TypeID": "AppConfig-ABCDEFG"
            },
            "lives": [
                {
                    "liveID": "AppConfig-1",
                    "config": {
                        "name": "My App"
                    }
                }
            ]
        },
        {
            "metaData": {
                "name": "DBConfig Brick",
                "TypeID": "DBConfig-HIJKLMN"
            },
            "lives": [
                {
                    "liveID": "DBConfig-1",
                    "config": {
                        "name": "My DB"
                    }
                }
            ]
        }
    ]
}
	`)
	if err != nil {
		panic(err)
	}

	brick.SetLiveIDConstraint(false)

	err = brick.AddConfigFile(configFile.Name())
	if err != nil {
		panic(err)
	}

	// 获取并使用组件
	userService := brick.GetOrCreate[*UserService]()
	userService.DB.Connect()
	userService.MyLogger.Log("User service started")

	orderService := brick.GetOrCreate[*OrderService]()
	orderService.DB.Connect()

	configService := brick.GetOrCreate[*ConfigService]()
	fmt.Println("AppConfig Name:", configService.AppConfig.GetName())
	fmt.Println("DBConfig Name:", configService.DBConfig.GetName())

	// 获取并使用克隆的实例
	cloneService := brick.GetOrCreate[*CloneService]()
	fmt.Println("CloneService Logger Prefix:", cloneService.Logger.Prefix)
	fmt.Println("CloneService Logger2 Prefix:", cloneService.Logger2.Prefix)
	fmt.Println("CloneService ConfigService AppConfig Name:", cloneService.ConfigService.AppConfig.GetName())
	fmt.Println("CloneService ConfigService DBConfig Name:", cloneService.ConfigService.DBConfig.GetName())

	// 修改克隆的实例
	cloneService.Logger.Prefix = "Cloned Logger"
	cloneService.Logger2.Prefix = "Cloned Logger2"
	cloneService.ConfigService.AppConfig.(*AppConfig).Name = "Cloned AppConfig"
	cloneService.ConfigService.DBConfig.(*DBConfig).Name = "Cloned DBConfig"

	// 再次获取克隆的实例，查看是否被修改
	cloneService2 := brick.GetOrCreate[*CloneService]()
	fmt.Println("CloneService2 Logger Prefix:", cloneService2.Logger.Prefix)
	fmt.Println("CloneService2 Logger2 Prefix:", cloneService2.Logger2.Prefix)
	fmt.Println("CloneService2 ConfigService AppConfig Name:", cloneService2.ConfigService.AppConfig.GetName())
	fmt.Println("CloneService2 ConfigService DBConfig Name:", cloneService2.ConfigService.DBConfig.GetName())

	// 获取并测试非指针类型的单例
	nonPtrSingleton := brick.GetOrCreate[*NonPtrSingleton]()
	nonPtrSingleton.Counter++
	fmt.Println("NonPtrSingleton Counter:", nonPtrSingleton.Counter)
	nonPtrSingleton.Logger.Log("NonPtrSingleton Counter: " + fmt.Sprint(nonPtrSingleton.Counter))

	nonPtrSingleton2 := brick.GetOrCreate[*NonPtrSingleton]()
	nonPtrSingleton2.Counter++
	fmt.Println("NonPtrSingleton2 Counter:", nonPtrSingleton2.Counter)
	nonPtrSingleton2.Logger.Log("NonPtrSingleton2 Counter: " + fmt.Sprint(nonPtrSingleton2.Counter))

	nonPtrSingleton3 := brick.GetOrCreate[*NonPtrSingleton]()
	nonPtrSingleton3.Counter++
	fmt.Println("NonPtrSingleton3 Counter:", nonPtrSingleton3.Counter)
	nonPtrSingleton3.Logger.Log("NonPtrSingleton3 Counter: " + fmt.Sprint(nonPtrSingleton3.Counter))

	nonPtrSingleton4 := brick.GetOrCreate[*NonPtrSingleton]()
	nonPtrSingleton4.Counter++
	fmt.Println("NonPtrSingleton4 Counter:", nonPtrSingleton4.Counter)
	nonPtrSingleton4.Logger.Log("NonPtrSingleton4 Counter: " + fmt.Sprint(nonPtrSingleton4.Counter))
}

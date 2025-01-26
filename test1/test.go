package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/doraemonkeys/brick"
)

// IDataBase 数据库接口
type IDataBase interface {
	Connect() error
	Query(string) string
}

// Logger 组件
type Logger struct {
	Prefix   string `json:"prefix"`
	LogLevel string `json:"logLevel"`
}

func (l *Logger) BrickTypeID() string {
	return "LoggerTypeID-XU8WQNK1F65QVH4OQ5XY"
}

func (l *Logger) NewBrick(jsonConf []byte) brick.Brick {
	var newLogger = &Logger{}
	if jsonConf != nil {
		err := json.Unmarshal(jsonConf, newLogger)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal logger config: %w", err))
		}
	} else {
		// 默认配置
		newLogger.Prefix = "[Default]"
		newLogger.LogLevel = "INFO"
	}
	return newLogger
}

// Database 组件
type Database struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
}

func (db *Database) BrickTypeID() string {
	return "DatabaseTypeID-9CD5C79CDPWBCD8462A8"
}

func (db *Database) NewBrick(jsonConf []byte) brick.Brick {
	var newDB = &Database{}
	if jsonConf != nil {
		err := json.Unmarshal(jsonConf, newDB)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal database config: %w", err))
		}
	}
	return newDB
}

func (db *Database) Connect() error {
	fmt.Printf("Connecting to %s database: %s@%s:%s/%s\n", db.Driver, db.Username, db.Host, db.Port, db.DBName)
	// 模拟连接数据库
	return nil
}

func (db *Database) Query(sql string) string {
	fmt.Printf("Executing query: %s\n", sql)
	// 模拟执行查询
	return "Query result"
}

// Config 组件 - 用于演示非 BrickNewer 组件
type Config struct {
	AppName string `json:"appName"`
}

func (c *Config) BrickTypeID() string {
	return "ConfigTypeID-961S959CDPWBCD8462A8"
}

func (c *Config) NewBrick(jsonConf []byte) brick.Brick {
	var newConfig = &Config{}
	if jsonConf != nil {
		err := json.Unmarshal(jsonConf, newConfig)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal config: %w", err))
		}
	}
	return newConfig
}

// UserService 组件
type UserService struct {
	Logger   *Logger   `brick:"LoggerTypeID-XU8WQNK1F65QVH4OQ5XY"` // 注入 Logger 组件
	Database *Database `brick:"MySQLDatabase"`                     // 注入 Database 组件，LiveID 为 MySQLDatabase
	Config   *Config   `brick:"ConfigTypeID-961S959CDPWBCD8462A8"` // 注入 Config 组件
}

func (us *UserService) BrickTypeID() string {
	return "UserServiceTypeID-789ABC123DEF456GHIJ"
}

func (us *UserService) GetUser(id int) string {
	us.Logger.Log(fmt.Sprintf("Getting user with ID: %d", id))
	result := us.Database.Query(fmt.Sprintf("SELECT * FROM users WHERE id = %d", id))
	return fmt.Sprintf("User data: %s, AppName: %s", result, us.Config.AppName)
}

// OrderService 组件
type OrderService struct {
	Logger   *Logger   `brick:"clone:OrderServiceLogger"` // 注入 Logger 组件的克隆，LiveID 为 OrderServiceLogger
	Database IDataBase `brick:"MySQLDatabase"`            // 注入 Database 组件，类型为 IDataBase 接口
}

func (os *OrderService) BrickTypeID() string {
	return "OrderServiceTypeID-KLM123NOP456QRS789"
}

func (os *OrderService) CreateOrder(order string) {
	os.Logger.Log(fmt.Sprintf("Creating order: %s", order))
	os.Database.Query(fmt.Sprintf("INSERT INTO orders (data) VALUES ('%s')", order))
}

func (l *Logger) Log(message string) {
	fmt.Printf("%s [%s] %s\n", l.Prefix, l.LogLevel, message)
}

func init() {
	// 设置环境变量，模拟从环境变量中读取配置
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("DB_USER", "root")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("DB_NAME", "mydatabase")

	// 注册组件
	brick.RegisterNewer[*Logger]()
	brick.RegisterNewer[*Database]()
	brick.RegisterNewer[*Config]()
	brick.Register[*UserService]()
	brick.Register[*OrderService]()
}

func main() {
	// 添加配置文件
	err := brick.AddConfigFile("bricks.json")
	if err != nil {
		panic(err)
	}

	// 获取 UserService 组件实例
	userService := brick.GetOrCreate[*UserService]()
	fmt.Println(userService.GetUser(1))

	// 获取 OrderService 组件实例
	orderService := brick.GetOrCreate[*OrderService]()
	orderService.CreateOrder("Some order data")

	fmt.Println("----------------------------------")

	// 手动注册LiveID
	brick.RegisterLiveIDType[*Database]("MySQLDatabase")
	orderService2 := brick.GetOrCreate[*OrderService]()
	orderService2.CreateOrder("Some order data2")
}

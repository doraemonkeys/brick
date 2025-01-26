package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/doraemonkeys/brick"
	"github.com/doraemonkeys/doraemon"
)

// Config
type LogConfig struct {
	Format string `json:"format"`
}

type DBConfig struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type ServerConfig struct {
	Port int `json:"port"`
}

// Bricks
// Logger Brick
type Logger struct {
	Format string `json:"format"`
}

func (l *Logger) BrickTypeID() string {
	return "Logger"
}

func (l *Logger) NewBrick(jsonConfig []byte) brick.Brick {
	var config LogConfig
	if jsonConfig != nil {
		err := json.Unmarshal(jsonConfig, &config)
		if err != nil {
			panic(fmt.Errorf("logger config unmarshal error: %w", err))
		}
	}
	return &Logger{
		Format: config.Format,
	}
}

func (l *Logger) Log(message string) {
	now := time.Now().Format(l.Format)
	fmt.Printf("[%s] %s\n", now, message)
}

// DB Brick
type DB struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

func (db *DB) BrickTypeID() string {
	return "DB"
}

func (db *DB) NewBrick(jsonConfig []byte) brick.Brick {
	var config DBConfig
	if jsonConfig != nil {
		err := json.Unmarshal(jsonConfig, &config)
		if err != nil {
			panic(fmt.Errorf("db config unmarshal error: %w", err))
		}
	}
	return &DB{
		Driver: config.Driver,
		DSN:    config.DSN,
	}
}

func (db *DB) Query(query string) {
	fmt.Printf("DB: %s, Query: %s\n", db.Driver, query)
}

// Redis Brick
type Redis struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

func (r *Redis) BrickTypeID() string {
	return "Redis"
}

func (r *Redis) NewBrick(jsonConfig []byte) brick.Brick {
	var config RedisConfig
	if jsonConfig != nil {
		err := json.Unmarshal(jsonConfig, &config)
		if err != nil {
			panic(fmt.Errorf("redis config unmarshal error: %w", err))
		}
	}
	return &Redis{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	}
}

func (r *Redis) Get(key string) {
	fmt.Printf("Redis Get: %s\n", key)
}

// Server Brick
type Server struct {
	Port    int         `json:"port"`
	Logger  *Logger     `brick:""`     // Inject Logger with the same typeID
	DB      *DB         `brick:"mydb"` // Inject DB with liveID "mydb"
	Redis   *Redis      `brick:""`     // Inject Redis with the same typeID
	Service *Service    `brick:""`     // Inject Service with the same typeID
	IDGen   IDGener     `brick:"UUIDGenerator"`
	Logger2 interface{} `brick:"logger2"` // Inject Logger with liveID "logger2"
	Logger3 interface{} `brick:"Logger"`  // Inject Logger with typeID "Logger"
}

func (s *Server) BrickTypeID() string {
	return "Server"
}

func (s *Server) NewBrick(jsonConfig []byte) brick.Brick {
	var config ServerConfig
	if jsonConfig != nil {
		err := json.Unmarshal(jsonConfig, &config)
		if err != nil {
			panic(fmt.Errorf("server config unmarshal error: %w", err))
		}
	}
	return &Server{
		Port: config.Port,
	}
}

func (s *Server) Start() {
	s.Logger.Log(fmt.Sprintf("Server started on port %d", s.Port))
	s.DB.Query("SELECT * FROM users")
	s.Redis.Get("mykey")
	s.Service.DoSomething()
	s.Logger.Log(fmt.Sprintf("IDGen type:%T ,value: %s", s.IDGen, s.IDGen.GenID()))
	s.Logger.Log(fmt.Sprintf("Logger2 type:%T ,value: %v", s.Logger2, s.Logger2))
	s.Logger.Log(fmt.Sprintf("Logger3 type:%T ,value: %v", s.Logger3, s.Logger3))
}

// Service Brick
type Service struct {
	Logger *Logger `brick:""` // Inject Logger with the same typeID
}

func (s *Service) BrickTypeID() string {
	return "Service"
}

func (s *Service) DoSomething() {
	s.Logger.Log("Service is doing something")
}

type IDGener interface {
	GenID() string
}

type UUIDGenerator struct {
}

func (g *UUIDGenerator) GenID() string {
	return doraemon.GenRandomAsciiString(10)
}

func (g *UUIDGenerator) BrickTypeID() string {
	return "UUIDGenerator"
}

func main() {
	// Add config file
	err := brick.AddConfigFile("config.json")
	if err != nil {
		panic(err)
	}

	// Register bricks
	brick.RegisterNewer[*Logger]()
	brick.RegisterNewer[*DB]()
	brick.RegisterNewer[*Redis]()
	brick.Register[*Service]()
	brick.RegisterNewer[*Server]()
	brick.Register[*UUIDGenerator]()
	brick.RegisterLiveIDType[*Logger]("logger2")

	// Get server instance and start
	server := brick.GetOrCreate[*Server]()
	server.Start()

	logger2 := brick.GetOrCreate[*Logger]("logger2")
	logger2.Log("hello logger2")

	uuidGen := brick.GetOrCreate[*UUIDGenerator]()
	fmt.Println(uuidGen.GenID())
}

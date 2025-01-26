package components

import (
	"encoding/json"
	"fmt"

	"github.com/doraemonkeys/brick"
)

// DatabaseConfig represents the configuration for the database.
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// Database implements Brick and represents a database connection.
type Database struct {
	Config DatabaseConfig
}

// BrickTypeID implements the Brick interface.
func (d *Database) BrickTypeID() string {
	return "Database"
}

// NewBrick implements the BrickNewer interface.
func (d *Database) NewBrick(jsonConfig []byte) brick.Brick {
	config := DatabaseConfig{}
	if len(jsonConfig) > 0 {
		if err := json.Unmarshal(jsonConfig, &config); err != nil {
			panic(err)
		}
	}
	return &Database{Config: config}
}

// Connect establishes a database connection (simulated).
func (d *Database) Connect() {
	fmt.Printf("Database connection established to: %s:%d/%s\n", d.Config.Host, d.Config.Port, d.Config.Database)
}

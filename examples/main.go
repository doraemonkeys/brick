package main

import (
	"fmt"
	"os"

	"github.com/doraemonkeys/brick"
	"github.com/doraemonkeys/brick/examples/components"
	"github.com/doraemonkeys/doraemon"
)

func main() {
	// Load configurations from files
	brick.AddConfigFile("config/app.yaml")
	brick.AddConfigFile("config/mysql.yaml")
	os.Setenv("API_KEY", "your_api_key")
	os.Setenv("MYSQL_HOST", "127.0.0.1")
	os.Setenv("MYSQL_USER", "root")
	os.Setenv("MYSQL_PASSWORD", "123456")

	// Register brick types and their factories.
	brick.RegisterNewer[*components.Database]()
	brick.RegisterNewer[*components.UserService]()
	brick.RegisterNewer[*components.ProductService]()
	brick.RegisterNewer[*components.ConfigCenter]()
	brick.RegisterNewer[*components.CacheService]()

	configCenter := brick.GetOrCreate[*components.ConfigCenter]("ConfigCenter")
	cache := brick.GetOrCreate[*components.CacheService]("CacheService")

	fmt.Println("app name from config:", configCenter.GetConfig("app_name"))
	fmt.Println("api key from config:", configCenter.GetConfig("api_key"))
	fmt.Println("cache enabled config from config:", cache.GetCache("cache_enabled"))
	fmt.Println("cache type config from config:", cache.GetCache("cache_type"))

	// Get instances of bricks, dependencies will be automatically injected
	userService := brick.GetOrCreate[*components.UserService]("UserService")
	productService := brick.GetOrCreate[*components.ProductService]("ProductService")

	// Use the services
	userService.GetUser(1)
	productService.GetProduct(10)

	fmt.Println(doraemon.GenRandomString("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 20))
}

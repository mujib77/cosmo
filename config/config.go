package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	RefreshRate int
}

// Load reads the configuration from environment variables and returns a Config.
func Load() Config {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("no .env file found, using enviornment variables")
	}
    
	return Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RefreshRate: 2,
}
}
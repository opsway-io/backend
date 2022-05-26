package influxdb

import (
	// Used to load .env files for environment variables
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

// Config contains environment configurable variables
type Config struct {
	Host  string `required:"true"`
	Port  uint32 `required:"true"`
	Token string `required:"true"`
}

// LoadEnvConfig is the constructor for EnvConfig
func loadEnvConfig() (*Config, error) {
	var c Config
	err := envconfig.Process("influxdb", &c)
	return &c, err
}

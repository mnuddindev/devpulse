package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	ServerConfig `mapstructure:",squash"`
	Postgres     `mapstructure:",squash"`
}

type ServerConfig struct {
	App       string `mapstructure:"APP"`
	Version   string `mapstructure:"VERSION"`
	Port      string `mapstructure:"PORT"`
	Status    string `mapstructure:"STATUS"`
	JWTSecret string `mapstructure:"JWT_SECRET"`
}

type Postgres struct {
	Host string `mapstructure:"DB_HOST"`
	Port string `mapstructure:"DB_PORT"`
	User string `mapstructure:"DB_USER"`
	Pass string `mapstructure:"DB_PASS"`
	Name string `mapstructure:"DB_NAME"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("app.env")
	viper.AddConfigPath("./")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshalling config, %s", err)
		return nil, err
	}

	return &config, nil
}

package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP        string
	Version    string
	Status     string
	Author     string
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	RedisAddr  string
	ServerAddr string
	JWTSecret  string
}

func LoadConfig() *Config {
	godotenv.Load()
	return &Config{
		APP:        os.Getenv("APP"),
		Version:    os.Getenv("VERSION"),
		Status:     os.Getenv("STATUS"),
		Author:     os.Getenv("AUTHOR"),
		DBHost:     os.Getenv("DB_HOST"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASS"),
		DBName:     os.Getenv("DB_NAME"),
		RedisAddr:  os.Getenv("REDIS_ADDR"),
		ServerAddr: os.Getenv("PORT"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
	}
}

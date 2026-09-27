package config

import "os"

type Config struct {
	DBUrl     string
	JWTSecret string
	RedisURL  string
}

func Load() *Config {
	return &Config{
		DBUrl:     os.Getenv("DATABASE_URL"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		RedisURL:  os.Getenv("REDIS_URL"),
	}
}

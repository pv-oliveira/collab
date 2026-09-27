package config

import (
	"os"
	"strings"
)

type Config struct {
	DBUrl          string
	JWTSecret      string
	RedisURL       string
	AllowedOrigins []string
}

func Load() *Config {
	return &Config{
		DBUrl:          os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AllowedOrigins: splitList(os.Getenv("ALLOWED_ORIGINS")),
	}
}

// splitList lê "a, b,c" como ["a" "b" "c"], ignorando itens vazios.
func splitList(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

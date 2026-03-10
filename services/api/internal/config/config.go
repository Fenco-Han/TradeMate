package config

import "os"
import "strconv"

type Config struct {
	APIPort        string
	JWTSecret      string
	JWTExpiresHour int
	MySQLDSN       string
}

func Load() Config {
	expiresHour := 24
	if value := os.Getenv("JWT_EXPIRES_HOURS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			expiresHour = parsed
		}
	}

	return Config{
		APIPort:        getEnv("API_PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me"),
		JWTExpiresHour: expiresHour,
		MySQLDSN:       getEnv("MYSQL_DSN", "trademate:trademate@tcp(localhost:3306)/trademate?parseTime=true"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

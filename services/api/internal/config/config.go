package config

import "os"
import "strconv"

type Config struct {
	APIPort               string
	JWTSecret             string
	JWTExpiresHour        int
	MySQLDSN              string
	AmazonAdsClientID     string
	AmazonAdsClientSecret string
	AmazonAdsRedirectURI  string
	AmazonAdsAPIBase      string
	AmazonAdsTokenURL     string
}

func Load() Config {
	expiresHour := 24
	if value := os.Getenv("JWT_EXPIRES_HOURS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			expiresHour = parsed
		}
	}

	return Config{
		APIPort:               getEnv("API_PORT", "8080"),
		JWTSecret:             getEnv("JWT_SECRET", "change-me"),
		JWTExpiresHour:        expiresHour,
		MySQLDSN:              getEnv("MYSQL_DSN", "trademate:trademate@tcp(localhost:3306)/trademate?parseTime=true"),
		AmazonAdsClientID:     getEnv("AMAZON_ADS_CLIENT_ID", ""),
		AmazonAdsClientSecret: getEnv("AMAZON_ADS_CLIENT_SECRET", ""),
		AmazonAdsRedirectURI:  getEnv("AMAZON_ADS_REDIRECT_URI", ""),
		AmazonAdsAPIBase:      getEnv("AMAZON_ADS_API_BASE", "https://advertising-api.amazon.com"),
		AmazonAdsTokenURL:     getEnv("AMAZON_ADS_TOKEN_URL", "https://api.amazon.com/auth/o2/token"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

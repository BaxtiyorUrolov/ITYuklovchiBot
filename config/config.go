package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/cast"
	"os"
)

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	Environment string

	BotToken  string
	InstaApi  string
	TikTokApi string

	LoggerLevel string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("error !", err.Error())
	}
	cfg := Config{}

	cfg.PostgresHost = cast.ToString(getOrReturnDefault("POSTGRES_HOST", "localhost"))
	cfg.PostgresPort = cast.ToString(getOrReturnDefault("POSTGRES_PORT", "5432"))
	cfg.PostgresUser = cast.ToString(getOrReturnDefault("POSTGRES_USER", "your user"))
	cfg.PostgresPassword = cast.ToString(getOrReturnDefault("POSTGRES_PASSWORD", "your password"))
	cfg.PostgresDB = cast.ToString(getOrReturnDefault("POSTGRES_DB", "your database"))

	cfg.Environment = cast.ToString(getOrReturnDefault("ENVIRONMENT", "development"))

	cfg.BotToken = cast.ToString(getOrReturnDefault("BOT_TOKEN", "your token"))
	cfg.InstaApi = cast.ToString(getOrReturnDefault("INSTA_API", "https://api.instagram.com"))
	cfg.TikTokApi = cast.ToString(getOrReturnDefault("TIK_TOK_API", "https://api.tiktok.com"))

	cfg.LoggerLevel = cast.ToString(getOrReturnDefault("LOGGER_LEVEL", "debug"))

	return cfg
}

func getOrReturnDefault(key string, defaultValue interface{}) interface{} {
	value := os.Getenv(key)
	if value != "" {
		return value
	}

	return defaultValue
}

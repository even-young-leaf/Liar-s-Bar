package config

import (
	"os"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Game     GameConfig
	AI       AIConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func (d DatabaseConfig) DSN() string {
	return d.User + ":" + d.Password + "@tcp(" + d.Host + ":" + d.Port + ")/" + d.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	ExpireTime time.Duration
}

type GameConfig struct {
	MaxPlayers      int
	MatchTimeout    time.Duration
	AIJoinTimeout    time.Duration
	AIFillTimeout    time.Duration
	ReconnectTimeout time.Duration
	MaxCardsPlay    int
}

type AIConfig struct {
	ServiceURL string
	Enabled    bool
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "mysql"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "liarsbar"),
			Password: getEnv("DB_PASSWORD", "liarsbar123"),
			DBName:   getEnv("DB_NAME", "liars_bar"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "redis:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "liars-bar-secret-key-2024"),
			ExpireTime: 24 * time.Hour,
		},
		Game: GameConfig{
			MaxPlayers:      4,
			MatchTimeout:    10 * time.Second,
			AIJoinTimeout:    10 * time.Second,
			AIFillTimeout:    1 * time.Second,
			ReconnectTimeout: 30 * time.Second,
			MaxCardsPlay:    3,
		},
		AI: AIConfig{
			ServiceURL: getEnv("AI_SERVICE_URL", "http://ai-service:8000"),
			Enabled:    getEnv("AI_ENABLED", "true") == "true",
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

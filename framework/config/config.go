package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config is the central typed configuration struct.
// Embed or extend it in your app's own AppConfig.
type Config struct {
	App  AppConfig
	DB   DBConfig
	Mail MailConfig
}

type AppConfig struct {
	Name  string
	Env   string // local | production | testing
	Debug bool
	URL   string
	Port  string
	Key   string
}

type DBConfig struct {
	Driver   string
	Host     string
	Port     string
	Database string
	Username string
	Password string
}

type MailConfig struct {
	Driver string
	Host   string
	Port   string
	From   string
}

// Load reads .env (if present) and populates a Config from environment variables.
// Call once at bootstrap: cfg := config.Load()
func Load(envFiles ...string) *Config {
	files := envFiles
	if len(files) == 0 {
		files = []string{".env"}
	}
	// Non-fatal: .env may not exist in production
	_ = godotenv.Load(files...)

	return &Config{
		App: AppConfig{
			Name:  env("APP_NAME", "GoLaravel"),
			Env:   env("APP_ENV", "local"),
			Debug: envBool("APP_DEBUG", true),
			URL:   env("APP_URL", "http://localhost"),
			Port:  env("APP_PORT", "8000"),
			Key:   env("APP_KEY", ""),
		},
		DB: DBConfig{
			Driver:   env("DB_DRIVER", "mysql"),
			Host:     env("DB_HOST", "127.0.0.1"),
			Port:     env("DB_PORT", "3306"),
			Database: env("DB_DATABASE", ""),
			Username: env("DB_USERNAME", "root"),
			Password: env("DB_PASSWORD", ""),
		},
		Mail: MailConfig{
			Driver: env("MAIL_DRIVER", "smtp"),
			Host:   env("MAIL_HOST", ""),
			Port:   env("MAIL_PORT", "587"),
			From:   env("MAIL_FROM_ADDRESS", ""),
		},
	}
}

// Get returns a raw env value, falling back to defaultVal.
func Get(key, defaultVal string) string {
	return env(key, defaultVal)
}

// GetInt returns an int env value.
func GetInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}

// GetBool returns a bool env value.
func GetBool(key string, defaultVal bool) bool {
	return envBool(key, defaultVal)
}

// ── helpers ─────────────────────────────────────────────────────────────────

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

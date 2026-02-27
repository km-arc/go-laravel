package config_test

import (
	"os"
	"testing"

	"github.com/km-arc/go-laravel"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func setEnv(t *testing.T, key, val string) {
	t.Helper()
	t.Setenv(key, val) // automatically restored after test
}

// ── Load ─────────────────────────────────────────────────────────────────────

func TestLoad_Defaults(t *testing.T) {
	// No env set → verify all defaults
	cfg := config.Load("testdata/empty.env")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"App.Name", cfg.App.Name, "GoLaravel"},
		{"App.Env", cfg.App.Env, "local"},
		{"App.Port", cfg.App.Port, "8000"},
		{"DB.Driver", cfg.DB.Driver, "mysql"},
		{"DB.Host", cfg.DB.Host, "127.0.0.1"},
		{"DB.Port", cfg.DB.Port, "3306"},
		{"DB.Username", cfg.DB.Username, "root"},
		{"Mail.Driver", cfg.Mail.Driver, "smtp"},
		{"Mail.Port", cfg.Mail.Port, "587"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestLoad_EnvOverridesDefaults(t *testing.T) {
	setEnv(t, "APP_NAME", "MyApp")
	setEnv(t, "APP_ENV", "production")
	setEnv(t, "APP_PORT", "9000")
	setEnv(t, "DB_DATABASE", "mydb")

	cfg := config.Load()

	if cfg.App.Name != "MyApp" {
		t.Errorf("App.Name: got %q want %q", cfg.App.Name, "MyApp")
	}
	if cfg.App.Env != "production" {
		t.Errorf("App.Env: got %q want %q", cfg.App.Env, "production")
	}
	if cfg.App.Port != "9000" {
		t.Errorf("App.Port: got %q want %q", cfg.App.Port, "9000")
	}
	if cfg.DB.Database != "mydb" {
		t.Errorf("DB.Database: got %q want %q", cfg.DB.Database, "mydb")
	}
}

func TestLoad_AppDebugTrue(t *testing.T) {
	setEnv(t, "APP_DEBUG", "true")
	cfg := config.Load()
	if !cfg.App.Debug {
		t.Error("expected App.Debug to be true")
	}
}

func TestLoad_AppDebugFalse(t *testing.T) {
	setEnv(t, "APP_DEBUG", "false")
	cfg := config.Load()
	if cfg.App.Debug {
		t.Error("expected App.Debug to be false")
	}
}

// ── Get / GetInt / GetBool ───────────────────────────────────────────────────

func TestGet_ReturnsValue(t *testing.T) {
	setEnv(t, "CUSTOM_KEY", "hello")
	if got := config.Get("CUSTOM_KEY", "default"); got != "hello" {
		t.Errorf("got %q want %q", got, "hello")
	}
}

func TestGet_ReturnsFallback(t *testing.T) {
	os.Unsetenv("MISSING_KEY")
	if got := config.Get("MISSING_KEY", "fallback"); got != "fallback" {
		t.Errorf("got %q want %q", got, "fallback")
	}
}

func TestGetInt_ReturnsInt(t *testing.T) {
	setEnv(t, "SOME_INT", "42")
	if got := config.GetInt("SOME_INT", 0); got != 42 {
		t.Errorf("got %d want %d", got, 42)
	}
}

func TestGetInt_ReturnsFallbackOnInvalid(t *testing.T) {
	setEnv(t, "SOME_INT", "notanint")
	if got := config.GetInt("SOME_INT", 99); got != 99 {
		t.Errorf("got %d want %d", got, 99)
	}
}

func TestGetBool_True(t *testing.T) {
	for _, val := range []string{"true", "1", "True", "TRUE"} {
		setEnv(t, "BOOL_KEY", val)
		if !config.GetBool("BOOL_KEY", false) {
			t.Errorf("expected true for %q", val)
		}
	}
}

func TestGetBool_False(t *testing.T) {
	setEnv(t, "BOOL_KEY", "false")
	if config.GetBool("BOOL_KEY", true) {
		t.Error("expected false")
	}
}

func TestGetBool_ReturnsFallbackOnInvalid(t *testing.T) {
	setEnv(t, "BOOL_KEY", "notabool")
	if config.GetBool("BOOL_KEY", true) != true {
		t.Error("expected fallback true")
	}
}

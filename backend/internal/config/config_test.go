package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("VALKEY_PASSWORD", "")
	t.Setenv("LOG_DEV", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	cfg := Load()
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("unexpected addr: %q", cfg.HTTP.Addr)
	}
	if cfg.Postgres.DSN() == "" {
		t.Fatal("expected dsn")
	}
	if cfg.OTel.ServiceName != "licenseiq" {
		t.Fatalf("unexpected service name: %q", cfg.OTel.ServiceName)
	}
}

func TestValidate(t *testing.T) {
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}

	cfg.HTTP.Addr = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty addr")
	}

	cfg = Load()
	cfg.Log.Level = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestDSNIncludesFields(t *testing.T) {
	cfg := PostgresConfig{
		Host:     "db.example",
		Port:     5432,
		User:     "alice",
		Password: "secret",
		Database: "licenseiq",
		SSLMode:  "disable",
	}
	dsn := cfg.DSN()
	for _, want := range []string{"postgres://alice:secret@", "db.example:5432", "/licenseiq", "sslmode=disable"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn %q missing %q", dsn, want)
		}
	}
}

func TestGetEnvDurationFallback(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")
	if got := getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second); got != 5*time.Second {
		t.Fatalf("expected fallback, got %v", got)
	}
}

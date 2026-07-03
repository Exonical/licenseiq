package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Valkey   ValkeyConfig
	Log      LogConfig
	OTel     OTelConfig
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

type ValkeyConfig struct {
	Addr     string
	Password string
}

type LogConfig struct {
	Level string
	Dev   bool
}

type OTelConfig struct {
	Endpoint    string
	ServiceName string
}

func Load() Config {
	cfg := Config{
		HTTP: HTTPConfig{
			Addr:            getEnv("HTTP_ADDR", ":8080"),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			User:     getEnv("POSTGRES_USER", "licenseiq"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: getEnv("POSTGRES_DATABASE", "licenseiq"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Valkey: ValkeyConfig{
			Addr:     getEnv("VALKEY_ADDR", "localhost:6379"),
			Password: os.Getenv("VALKEY_PASSWORD"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			Dev:   getEnvBool("LOG_DEV", false),
		},
		OTel: OTelConfig{
			Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			ServiceName: getEnv("OTEL_SERVICE_NAME", "licenseiq"),
		},
	}
	return cfg
}

func (c Config) Validate() error {
	if err := validateAddr(c.HTTP.Addr); err != nil {
		return fmt.Errorf("http addr: %w", err)
	}
	if c.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("http read timeout must be positive")
	}
	if c.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("http write timeout must be positive")
	}
	if c.HTTP.ShutdownTimeout <= 0 {
		return fmt.Errorf("http shutdown timeout must be positive")
	}
	if err := validatePort(c.Postgres.Port); err != nil {
		return fmt.Errorf("postgres port: %w", err)
	}
	if strings.TrimSpace(c.Postgres.Host) == "" {
		return fmt.Errorf("postgres host is required")
	}
	if strings.TrimSpace(c.Postgres.User) == "" {
		return fmt.Errorf("postgres user is required")
	}
	if strings.TrimSpace(c.Postgres.Database) == "" {
		return fmt.Errorf("postgres database is required")
	}
	if strings.TrimSpace(c.Postgres.SSLMode) == "" {
		return fmt.Errorf("postgres sslmode is required")
	}
	if err := validateAddr(c.Valkey.Addr); err != nil {
		return fmt.Errorf("valkey addr: %w", err)
	}
	if strings.TrimSpace(c.Log.Level) == "" {
		return fmt.Errorf("log level is required")
	}
	if err := validateLogLevel(c.Log.Level); err != nil {
		return fmt.Errorf("log level: %w", err)
	}
	if strings.TrimSpace(c.OTel.ServiceName) == "" {
		return fmt.Errorf("otel service name is required")
	}
	return nil
}

func (c PostgresConfig) DSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.Database,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func validateAddr(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value is required")
	}
	if _, err := net.ResolveTCPAddr("tcp", value); err != nil {
		return err
	}
	return nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("must be between 1 and 65535")
	}
	return nil
}

func validateLogLevel(level string) error {
	var parsed zapcore.Level
	return parsed.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(level))))
}

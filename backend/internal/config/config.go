package config

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	HTTP         HTTPConfig
	Postgres     PostgresConfig
	Valkey       ValkeyConfig
	Log          LogConfig
	OTel         OTelConfig
	Auth         AuthConfig
	FeatureFlags FeatureFlagsConfig
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

type AuthConfig struct {
	OIDC      OIDCConfig
	Bootstrap BootstrapConfig
}

type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	Audience     string
	Scopes       []string
	RoleClaim    string
	RoleMappings map[string]domain.Role
	DefaultRole  domain.Role
}

type BootstrapConfig struct {
	AdminEmail       string
	AdminDisplayName string
	AdminAPIKey      string
	AdminAPIKeyName  string
}

type FeatureFlagsConfig struct {
	Overrides map[string]bool
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
		Auth: AuthConfig{
			OIDC: OIDCConfig{
				IssuerURL:    strings.TrimSpace(os.Getenv("OIDC_ISSUER_URL")),
				ClientID:     strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")),
				ClientSecret: strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
				Audience:     strings.TrimSpace(os.Getenv("OIDC_AUDIENCE")),
				Scopes:       splitCSV(getEnv("OIDC_SCOPES", "openid,profile,email")),
				RoleClaim:    getEnv("OIDC_ROLE_CLAIM", "groups"),
				RoleMappings: parseRoleMappings(os.Getenv("OIDC_ROLE_MAPPINGS")),
				DefaultRole:  parseRole(getEnv("OIDC_DEFAULT_ROLE", string(domain.RoleViewer))),
			},
			Bootstrap: BootstrapConfig{
				AdminEmail:       strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")),
				AdminDisplayName: strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_DISPLAY_NAME")),
				AdminAPIKey:      strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_API_KEY")),
				AdminAPIKeyName:  getEnv("BOOTSTRAP_ADMIN_API_KEY_NAME", "bootstrap-admin"),
			},
		},
		FeatureFlags: FeatureFlagsConfig{Overrides: parseFeatureFlagOverrides(os.Environ())},
	}
	if cfg.Auth.Bootstrap.AdminDisplayName == "" && cfg.Auth.Bootstrap.AdminEmail != "" {
		cfg.Auth.Bootstrap.AdminDisplayName = cfg.Auth.Bootstrap.AdminEmail
	}
	if len(cfg.Auth.OIDC.RoleMappings) == 0 {
		cfg.Auth.OIDC.RoleMappings = map[string]domain.Role{}
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
	if err := c.Auth.Validate(); err != nil {
		return err
	}
	return nil
}

func (c AuthConfig) Validate() error {
	if err := c.OIDC.Validate(); err != nil {
		return err
	}
	if c.Bootstrap.AdminEmail != "" {
		if _, err := parseEmail(c.Bootstrap.AdminEmail); err != nil {
			return fmt.Errorf("bootstrap admin email: %w", err)
		}
	}
	if c.Bootstrap.AdminAPIKeyName == "" {
		return fmt.Errorf("bootstrap admin api key name is required")
	}
	return nil
}

func (c OIDCConfig) Validate() error {
	if c.IssuerURL == "" {
		return nil
	}
	if c.ClientID == "" {
		return fmt.Errorf("oidc client id is required when issuer url is configured")
	}
	if c.RoleClaim == "" {
		return fmt.Errorf("oidc role claim is required")
	}
	if err := c.DefaultRole.Validate(); err != nil {
		return fmt.Errorf("oidc default role: %w", err)
	}
	for claimValue, role := range c.RoleMappings {
		if claimValue == "" {
			return fmt.Errorf("oidc role mapping claim value is required")
		}
		if err := role.Validate(); err != nil {
			return fmt.Errorf("oidc role mapping for %q: %w", claimValue, err)
		}
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

func (c OIDCConfig) ScopesOrDefault() []string {
	if len(c.Scopes) == 0 {
		return []string{"openid", "profile", "email"}
	}
	return append([]string(nil), c.Scopes...)
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

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseRoleMappings(value string) map[string]domain.Role {
	out := map[string]domain.Role{}
	for _, item := range splitCSV(value) {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		claimValue := strings.TrimSpace(parts[0])
		role := parseRole(parts[1])
		if claimValue != "" {
			out[claimValue] = role
		}
	}
	return out
}

func parseRole(value string) domain.Role {
	role, err := domain.ParseRole(value)
	if err != nil {
		return domain.RoleViewer
	}
	return role
}

func parseFeatureFlagOverrides(values []string) map[string]bool {
	overrides := map[string]bool{}
	for _, entry := range values {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(key, "FEATUREFLAG_") {
			continue
		}
		normalizedKey := normalizeFeatureFlagKey(strings.TrimPrefix(key, "FEATUREFLAG_"))
		if normalizedKey == "" {
			continue
		}
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		overrides[normalizedKey] = parsed
	}
	return overrides
}

func normalizeFeatureFlagKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func parseEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("value is required")
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return "", err
	}
	return value, nil
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

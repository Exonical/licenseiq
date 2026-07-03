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
	HTTP          HTTPConfig
	Postgres      PostgresConfig
	Valkey        ValkeyConfig
	Log           LogConfig
	OTel          OTelConfig
	Auth          AuthConfig
	FeatureFlags  FeatureFlagsConfig
	Notifications NotificationsConfig
	Jira          JiraConfig
	Workers       WorkersConfig
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

type NotificationsConfig struct {
	HTTPTimeout time.Duration
	SMTP        SMTPNotificationsConfig
	Slack       SlackNotificationsConfig
	Teams       TeamsNotificationsConfig
	Webhooks    WebhookNotificationsConfig
}

type JiraConfig struct {
	Enabled       bool
	BaseURL       string
	Deployment    string
	Email         string
	APIToken      string
	PersonalToken string
	ProjectKey    string
	IssueType     string
	HTTPTimeout   time.Duration
}

type WorkersConfig struct {
	Enabled     bool
	Timeout     time.Duration
	Renewals    RenewalReminderWorkerConfig
	Maintenance MaintenanceWorkerConfig
	JiraSync    JiraSyncWorkerConfig
}

type RenewalReminderWorkerConfig struct {
	Enabled  bool
	Interval time.Duration
}

type MaintenanceWorkerConfig struct {
	Enabled  bool
	Interval time.Duration
}

type JiraSyncWorkerConfig struct {
	Enabled    bool
	Interval   time.Duration
	WindowDays int
}

type SMTPNotificationsConfig struct {
	Enabled    bool
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	Recipients []string
	TLSMode    string
}

type SlackNotificationsConfig struct {
	WebhookURL string
}

type TeamsNotificationsConfig struct {
	WebhookURL string
}

type WebhookNotificationsConfig struct {
	URLs []string
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
		Notifications: NotificationsConfig{
			HTTPTimeout: getEnvDuration("NOTIFICATIONS_HTTP_TIMEOUT", 10*time.Second),
			SMTP: SMTPNotificationsConfig{
				Enabled:    getEnvBool("NOTIFICATIONS_SMTP_ENABLED", false),
				Host:       strings.TrimSpace(os.Getenv("NOTIFICATIONS_SMTP_HOST")),
				Port:       getEnvInt("NOTIFICATIONS_SMTP_PORT", 587),
				Username:   strings.TrimSpace(os.Getenv("NOTIFICATIONS_SMTP_USERNAME")),
				Password:   os.Getenv("NOTIFICATIONS_SMTP_PASSWORD"),
				From:       strings.TrimSpace(os.Getenv("NOTIFICATIONS_SMTP_FROM")),
				Recipients: splitCSV(os.Getenv("NOTIFICATIONS_SMTP_RECIPIENTS")),
				TLSMode:    strings.ToLower(strings.TrimSpace(getEnv("NOTIFICATIONS_SMTP_TLS_MODE", "starttls"))),
			},
			Slack:    SlackNotificationsConfig{WebhookURL: strings.TrimSpace(os.Getenv("NOTIFICATIONS_SLACK_WEBHOOK_URL"))},
			Teams:    TeamsNotificationsConfig{WebhookURL: strings.TrimSpace(os.Getenv("NOTIFICATIONS_TEAMS_WEBHOOK_URL"))},
			Webhooks: WebhookNotificationsConfig{URLs: splitCSV(os.Getenv("NOTIFICATIONS_WEBHOOK_URLS"))},
		},
		Jira: JiraConfig{
			Enabled:       getEnvBool("JIRA_ENABLED", false),
			BaseURL:       strings.TrimSpace(os.Getenv("JIRA_BASE_URL")),
			Deployment:    strings.ToLower(getEnv("JIRA_DEPLOYMENT", "cloud")),
			Email:         strings.TrimSpace(os.Getenv("JIRA_EMAIL")),
			APIToken:      strings.TrimSpace(os.Getenv("JIRA_API_TOKEN")),
			PersonalToken: strings.TrimSpace(getEnv("JIRA_PERSONAL_ACCESS_TOKEN", os.Getenv("JIRA_PAT"))),
			ProjectKey:    strings.TrimSpace(os.Getenv("JIRA_PROJECT_KEY")),
			IssueType:     strings.TrimSpace(os.Getenv("JIRA_ISSUE_TYPE")),
			HTTPTimeout:   getEnvDuration("JIRA_HTTP_TIMEOUT", 10*time.Second),
		},
		Workers: WorkersConfig{
			Enabled: getEnvBool("WORKERS_ENABLED", true),
			Timeout: getEnvDuration("WORKERS_TIMEOUT", 10*time.Minute),
			Renewals: RenewalReminderWorkerConfig{
				Enabled:  getEnvBool("WORKERS_RENEWAL_REMINDERS_ENABLED", true),
				Interval: getEnvDuration("WORKERS_RENEWAL_REMINDERS_INTERVAL", 24*time.Hour),
			},
			Maintenance: MaintenanceWorkerConfig{
				Enabled:  getEnvBool("WORKERS_MAINTENANCE_ENABLED", true),
				Interval: getEnvDuration("WORKERS_MAINTENANCE_INTERVAL", time.Hour),
			},
			JiraSync: JiraSyncWorkerConfig{
				Enabled:    getEnvBool("WORKERS_JIRA_SYNC_ENABLED", true),
				Interval:   getEnvDuration("WORKERS_JIRA_SYNC_INTERVAL", 24*time.Hour),
				WindowDays: getEnvInt("WORKERS_JIRA_SYNC_WINDOW_DAYS", 90),
			},
		},
	}
	if cfg.Auth.Bootstrap.AdminDisplayName == "" && cfg.Auth.Bootstrap.AdminEmail != "" {
		cfg.Auth.Bootstrap.AdminDisplayName = cfg.Auth.Bootstrap.AdminEmail
	}
	if len(cfg.Auth.OIDC.RoleMappings) == 0 {
		cfg.Auth.OIDC.RoleMappings = map[string]domain.Role{}
	}
	if cfg.Notifications.HTTPTimeout <= 0 {
		cfg.Notifications.HTTPTimeout = 10 * time.Second
	}
	if cfg.Notifications.SMTP.TLSMode == "" {
		cfg.Notifications.SMTP.TLSMode = "starttls"
	}
	if cfg.Jira.Deployment == "" {
		cfg.Jira.Deployment = "cloud"
	}
	if cfg.Jira.HTTPTimeout <= 0 {
		cfg.Jira.HTTPTimeout = 10 * time.Second
	}
	if cfg.Workers.Timeout <= 0 {
		cfg.Workers.Timeout = 10 * time.Minute
	}
	if cfg.Workers.Renewals.Interval <= 0 {
		cfg.Workers.Renewals.Interval = 24 * time.Hour
	}
	if cfg.Workers.Maintenance.Interval <= 0 {
		cfg.Workers.Maintenance.Interval = time.Hour
	}
	if cfg.Workers.JiraSync.Interval <= 0 {
		cfg.Workers.JiraSync.Interval = 24 * time.Hour
	}
	if cfg.Workers.JiraSync.WindowDays <= 0 {
		cfg.Workers.JiraSync.WindowDays = 90
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
	if err := c.Notifications.Validate(); err != nil {
		return err
	}
	if c.Jira.Enabled {
		if _, err := url.ParseRequestURI(c.Jira.BaseURL); err != nil {
			return fmt.Errorf("jira base url: %w", err)
		}
		switch c.Jira.Deployment {
		case "cloud", "datacenter":
		default:
			return fmt.Errorf("jira deployment must be cloud or datacenter")
		}
		if c.Jira.ProjectKey == "" {
			return fmt.Errorf("jira project key is required")
		}
		if c.Jira.IssueType == "" {
			return fmt.Errorf("jira issue type is required")
		}
		switch c.Jira.Deployment {
		case "cloud":
			if c.Jira.Email == "" || c.Jira.APIToken == "" {
				return fmt.Errorf("jira cloud email and api token are required")
			}
		case "datacenter":
			if c.Jira.PersonalToken == "" {
				return fmt.Errorf("jira datacenter personal access token is required")
			}
		}
	}
	if err := c.Workers.Validate(); err != nil {
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

func (c NotificationsConfig) Validate() error {
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("notifications http timeout must be positive")
	}
	if c.SMTP.Enabled {
		if strings.TrimSpace(c.SMTP.Host) == "" {
			return fmt.Errorf("notifications smtp host is required when enabled")
		}
		if c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
			return fmt.Errorf("notifications smtp port must be between 1 and 65535")
		}
		if strings.TrimSpace(c.SMTP.From) == "" {
			return fmt.Errorf("notifications smtp from is required when enabled")
		}
		if len(c.SMTP.Recipients) == 0 {
			return fmt.Errorf("notifications smtp recipients are required when enabled")
		}
		switch c.SMTP.TLSMode {
		case "", "off", "starttls", "implicit":
		default:
			return fmt.Errorf("notifications smtp tls mode must be off, starttls, or implicit")
		}
	}
	return nil
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

func (c WorkersConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("workers timeout must be positive")
	}
	if c.Enabled {
		if c.Renewals.Enabled && c.Renewals.Interval <= 0 {
			return fmt.Errorf("workers renewal reminder interval must be positive")
		}
		if c.Maintenance.Enabled && c.Maintenance.Interval <= 0 {
			return fmt.Errorf("workers maintenance interval must be positive")
		}
		if c.JiraSync.Enabled && c.JiraSync.Interval <= 0 {
			return fmt.Errorf("workers jira sync interval must be positive")
		}
		if c.JiraSync.Enabled && c.JiraSync.WindowDays <= 0 {
			return fmt.Errorf("workers jira sync window days must be positive")
		}
	}
	return nil
}

package featureflags

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	ffclient "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
	"github.com/thomaspoignant/go-feature-flag/retriever"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v2"
)

const defaultPollingInterval = 5 * time.Second

type Manager struct {
	repo      domain.FeatureFlagRepository
	overrides map[string]bool
	logger    *zap.Logger
}

type RepositoryRetriever struct {
	repo domain.FeatureFlagRepository
}

type goffFlag struct {
	Variations  map[string]bool `yaml:"variations"`
	Targeting   []goffTarget    `yaml:"targeting,omitempty"`
	DefaultRule goffRule        `yaml:"defaultRule"`
}

type goffTarget struct {
	Name       string         `yaml:"name"`
	Query      string         `yaml:"query"`
	Variation  string         `yaml:"variation,omitempty"`
	Percentage map[string]int `yaml:"percentage,omitempty"`
}

type goffRule struct {
	Name       string         `yaml:"name"`
	Variation  string         `yaml:"variation,omitempty"`
	Percentage map[string]int `yaml:"percentage,omitempty"`
}

func NewManager(ctx context.Context, cfg config.FeatureFlagsConfig, repo domain.FeatureFlagRepository, logger *zap.Logger) (*Manager, error) {
	if repo == nil {
		return nil, fmt.Errorf("feature flag repository is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if err := ffclient.Init(ffclient.Config{
		Context:                 ctx,
		Retriever:               &RepositoryRetriever{repo: repo},
		PollingInterval:         defaultPollingInterval,
		EnablePollingJitter:     true,
		StartWithRetrieverError: true,
		FileFormat:              "yaml",
	}); err != nil {
		return nil, err
	}
	return &Manager{repo: repo, overrides: normalizeOverrides(cfg.Overrides), logger: logger}, nil
}

func (m *Manager) Close() {
	ffclient.Close()
}

func (m *Manager) Evaluate(ctx context.Context, key string, defaultValue bool) bool {
	if m == nil {
		return defaultValue
	}
	if override, ok := m.overrideFor(key); ok {
		return override
	}
	value, err := ffclient.BoolVariation(key, evaluationContext(ctx), defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (m *Manager) EvaluateAll(ctx context.Context) (map[string]bool, error) {
	if m == nil {
		return map[string]bool{}, nil
	}
	flags, err := listAllFlags(ctx, m.repo)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(flags)+len(m.overrides))
	for _, flag := range flags {
		result[flag.Key] = m.Evaluate(ctx, flag.Key, false)
	}
	for key := range m.overrides {
		if _, ok := result[key]; !ok {
			result[key] = m.Evaluate(ctx, key, false)
		}
	}
	return result, nil
}

func RequireFlag(manager *Manager, key string, defaultValue bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if manager == nil || !manager.Evaluate(ctx.Context(), key, defaultValue) {
			ctx.SetStatus(404)
			ctx.SetHeader("Content-Type", "application/json; charset=utf-8")
			_, _ = fmt.Fprint(ctx.BodyWriter(), `{"message":"not found"}`)
			return
		}
		next(ctx)
	}
}

func (r *RepositoryRetriever) Retrieve(ctx context.Context) ([]byte, error) {
	flags, err := listAllFlags(ctx, r.repo)
	if err != nil {
		return nil, err
	}
	payload := make(map[string]goffFlag, len(flags))
	now := time.Now().UTC()
	for _, flag := range flags {
		payload[flag.Key] = toGOFFFlag(flag, now)
	}
	if len(payload) == 0 {
		return []byte("{}\n"), nil
	}
	return yaml.Marshal(payload)
}

func (r *RepositoryRetriever) Shutdown(context.Context) error { return nil }
func (r *RepositoryRetriever) Status() retriever.Status       { return retriever.RetrieverReady }

func toGOFFFlag(flag domain.FeatureFlag, now time.Time) goffFlag {
	out := goffFlag{Variations: map[string]bool{"disabled": false, "enabled": true}}
	if !isActive(flag, now) || !flag.Enabled {
		out.DefaultRule = goffRule{Name: "defaultRule", Variation: "disabled"}
		return out
	}
	out.Targeting = append(out.Targeting, userTargetRules(flag.TargetUserIDs)...)
	out.Targeting = append(out.Targeting, roleTargetRules(flag.TargetRoles)...)
	out.DefaultRule = rolloutRule(flag.PercentageRollout)
	return out
}

func rolloutRule(percentage int) goffRule {
	switch {
	case percentage <= 0:
		return goffRule{Name: "defaultRule", Variation: "disabled"}
	case percentage >= 100:
		return goffRule{Name: "defaultRule", Variation: "enabled"}
	default:
		return goffRule{Name: "defaultRule", Percentage: map[string]int{"enabled": percentage, "disabled": 100 - percentage}}
	}
}

func userTargetRules(userIDs []uuid.UUID) []goffTarget {
	if len(userIDs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		parts = append(parts, fmt.Sprintf("%q", id.String()))
	}
	sort.Strings(parts)
	return []goffTarget{{Name: "userTarget", Query: fmt.Sprintf("targetingKey in [%s]", strings.Join(parts, ", ")), Variation: "enabled"}}
}

func roleTargetRules(roles []domain.Role) []goffTarget {
	if len(roles) == 0 {
		return nil
	}
	parts := make([]string, 0, len(roles))
	for _, role := range roles {
		parts = append(parts, fmt.Sprintf("%q", role.String()))
	}
	sort.Strings(parts)
	return []goffTarget{{Name: "roleTarget", Query: fmt.Sprintf("role in [%s]", strings.Join(parts, ", ")), Variation: "enabled"}}
}

func isActive(flag domain.FeatureFlag, now time.Time) bool {
	if flag.ScheduledEnableAt != nil && now.Before(flag.ScheduledEnableAt.UTC()) {
		return false
	}
	if flag.ScheduledDisableAt != nil && now.After(flag.ScheduledDisableAt.UTC()) {
		return false
	}
	return true
}

func evaluationContext(ctx context.Context) ffcontext.EvaluationContext {
	if principal, ok := auth.PrincipalFromContext(ctx); ok && principal.UserID != nil {
		builder := ffcontext.NewEvaluationContextBuilder(principal.UserID.String())
		builder.AddCustom("userId", principal.UserID.String())
		builder.AddCustom("role", principal.Role.String())
		if principal.APIKeyID != nil {
			builder.AddCustom("apiKeyId", principal.APIKeyID.String())
		}
		builder.AddCustom("isServiceAccount", principal.IsServiceAccount)
		return builder.Build()
	}
	reqCtx := app.RequestContextFromContext(ctx)
	key := strings.TrimSpace(reqCtx.SessionID)
	if key == "" {
		key = strings.TrimSpace(reqCtx.IPAddress)
	}
	if key == "" {
		key = "anonymous"
	}
	builder := ffcontext.NewEvaluationContextBuilder(key)
	builder.AddCustom("anonymous", true)
	if reqCtx.ActorUserID != nil {
		builder.AddCustom("userId", reqCtx.ActorUserID.String())
	}
	return builder.Build()
}

func listAllFlags(ctx context.Context, repo domain.FeatureFlagRepository) ([]domain.FeatureFlag, error) {
	if repo == nil {
		return nil, nil
	}
	const pageSize = 500
	out := make([]domain.FeatureFlag, 0)
	for offset := 0; ; offset += pageSize {
		batch, err := repo.List(ctx, domain.ListFilter{Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return out, nil
}

func normalizeOverrides(values map[string]bool) map[string]bool {
	if len(values) == 0 {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(values))
	for key, value := range values {
		out[normalizeFlagKey(key)] = value
	}
	return out
}

func (m *Manager) overrideFor(key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	value, ok := m.overrides[normalizeFlagKey(key)]
	return value, ok
}

func normalizeFlagKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

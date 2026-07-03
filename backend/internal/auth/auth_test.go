package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func testHashAPIKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type fakeIdentity struct {
	serviceAccount *domain.User
}

func (f *fakeIdentity) ListServiceAccounts(context.Context, domain.ListFilter) ([]domain.User, error) {
	return nil, nil
}
func (f *fakeIdentity) GetServiceAccount(context.Context, uuid.UUID) (*domain.User, error) {
	if f.serviceAccount == nil {
		return nil, domain.ErrNotFound
	}
	return f.serviceAccount, nil
}
func (f *fakeIdentity) CreateServiceAccount(context.Context, domain.User) (*domain.User, error) {
	return nil, errors.New("unexpected")
}
func (f *fakeIdentity) UpdateServiceAccount(context.Context, uuid.UUID, domain.User) (*domain.User, error) {
	return nil, errors.New("unexpected")
}
func (f *fakeIdentity) DeleteServiceAccount(context.Context, uuid.UUID) error {
	return errors.New("unexpected")
}
func (f *fakeIdentity) UpsertAuthenticatedUser(context.Context, domain.User) (*domain.User, error) {
	return nil, errors.New("unexpected")
}
func (f *fakeIdentity) ListAPIKeys(context.Context, uuid.UUID, domain.ListFilter) ([]domain.APIKey, error) {
	return nil, nil
}
func (f *fakeIdentity) GetAPIKey(context.Context, uuid.UUID) (*domain.APIKey, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeIdentity) GetAPIKeyByKeyID(context.Context, string) (*domain.APIKey, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeIdentity) CreateAPIKey(context.Context, domain.APIKey) (*domain.APIKey, string, error) {
	return nil, "", errors.New("unexpected")
}
func (f *fakeIdentity) CreateAPIKeyWithToken(context.Context, domain.APIKey, string) (*domain.APIKey, error) {
	return nil, errors.New("unexpected")
}
func (f *fakeIdentity) DeleteAPIKey(context.Context, uuid.UUID) error {
	return errors.New("unexpected")
}

type fakeUserRepo struct{}

func (fakeUserRepo) Create(context.Context, *domain.User) error { return nil }
func (fakeUserRepo) Get(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (fakeUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (fakeUserRepo) GetByExternalSubject(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (fakeUserRepo) Update(context.Context, *domain.User) error                     { return nil }
func (fakeUserRepo) Delete(context.Context, uuid.UUID) error                        { return nil }
func (fakeUserRepo) List(context.Context, domain.ListFilter) ([]domain.User, error) { return nil, nil }

type fakeAPIKeyRepo struct{ keys map[string]*domain.APIKey }

func (r *fakeAPIKeyRepo) Create(_ context.Context, key *domain.APIKey) error {
	if r.keys == nil {
		r.keys = map[string]*domain.APIKey{}
	}
	clone := *key
	r.keys[key.KeyID] = &clone
	return nil
}
func (r *fakeAPIKeyRepo) Get(_ context.Context, id uuid.UUID) (*domain.APIKey, error) {
	for _, key := range r.keys {
		if key.ID == id {
			clone := *key
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *fakeAPIKeyRepo) GetByKeyID(_ context.Context, keyID string) (*domain.APIKey, error) {
	key, ok := r.keys[keyID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *key
	return &clone, nil
}
func (r *fakeAPIKeyRepo) Update(_ context.Context, key *domain.APIKey) error {
	clone := *key
	r.keys[key.KeyID] = &clone
	return nil
}
func (r *fakeAPIKeyRepo) Delete(_ context.Context, id uuid.UUID) error {
	for keyID, key := range r.keys {
		if key.ID == id {
			delete(r.keys, keyID)
			return nil
		}
	}
	return domain.ErrNotFound
}
func (r *fakeAPIKeyRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.APIKey, error) {
	out := make([]domain.APIKey, 0, len(r.keys))
	for _, key := range r.keys {
		out = append(out, *key)
	}
	return out, nil
}

type fakeToken struct {
	claims map[string]any
	sub    string
	aud    []string
	exp    time.Time
}

func (t fakeToken) Claims(v any) error {
	data, err := json.Marshal(t.claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
func (t fakeToken) Subject() string    { return t.sub }
func (t fakeToken) Issuer() string     { return "https://issuer.example" }
func (t fakeToken) Audience() []string { return append([]string(nil), t.aud...) }
func (t fakeToken) Expiry() time.Time  { return t.exp }

type fakeVerifier struct {
	token Token
	err   error
}

func (v fakeVerifier) Verify(context.Context, string) (Token, error) { return v.token, v.err }

func TestPermissionMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		actor    domain.Role
		resource string
		action   string
		allowed  bool
	}{
		{name: "viewer read vendor", actor: domain.RoleViewer, resource: "vendors", action: "read", allowed: true},
		{name: "viewer write vendor", actor: domain.RoleViewer, resource: "vendors", action: "write", allowed: false},
		{name: "manager write vendor", actor: domain.RoleLicenseManager, resource: "vendors", action: "write", allowed: true},
		{name: "auditor read audit", actor: domain.RoleAuditor, resource: "audit_logs", action: "read", allowed: true},
		{name: "finance service account write", actor: domain.RoleFinance, resource: "service_accounts", action: "write", allowed: false},
		{name: "admin api keys", actor: domain.RoleAdministrator, resource: "api_keys", action: "admin", allowed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required, ok := RequiredRole(tt.resource, tt.action)
			if !ok {
				t.Fatalf("missing permission for %s/%s", tt.resource, tt.action)
			}
			if got := Allows(tt.actor, required); got != tt.allowed {
				t.Fatalf("expected %v, got %v", tt.allowed, got)
			}
		})
	}
}

func TestAPIKeyAuthentication(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	serviceUser := &domain.User{Base: domain.Base{ID: uuid.New()}, Email: "svc@example.com", Role: domain.RoleAdministrator, IsServiceAccount: true, Active: true}

	testCases := []struct {
		name      string
		prepare   func(*fakeAPIKeyRepo, *fakeIdentity)
		wantError bool
	}{
		{name: "valid", prepare: func(_ *fakeAPIKeyRepo, _ *fakeIdentity) {}, wantError: false},
		{name: "expired", prepare: func(repo *fakeAPIKeyRepo, _ *fakeIdentity) {
			expired := now.Add(-time.Hour)
			repo.keys["key123"].ExpiresAt = &expired
		}, wantError: true},
		{name: "inactive owner", prepare: func(repo *fakeAPIKeyRepo, ident *fakeIdentity) {
			repo.keys["key123"].ExpiresAt = nil
			ident.serviceAccount.Active = false
		}, wantError: true},
		{name: "unknown", prepare: func(repo *fakeAPIKeyRepo, _ *fakeIdentity) { delete(repo.keys, "key123") }, wantError: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ident := &fakeIdentity{serviceAccount: &domain.User{Base: serviceUser.Base, Email: serviceUser.Email, Role: serviceUser.Role, IsServiceAccount: true, Active: true}}
			keyRepo := &fakeAPIKeyRepo{keys: map[string]*domain.APIKey{}}
			key := &domain.APIKey{Base: domain.Base{ID: uuid.New()}, KeyID: "key123", OwnerUserID: ident.serviceAccount.ID, Name: "ci", HashedKey: testHashAPIKey("liq_key123.abcdefghijklmnopqrstuvwxyz0123456789")}
			keyRepo.keys[key.KeyID] = key
			tc.prepare(keyRepo, ident)
			manager := &Manager{identity: ident, users: fakeUserRepo{}, keys: keyRepo, logger: zap.NewNop()}
			principal, err := manager.authenticateAPIKey(context.Background(), "liq_key123.abcdefghijklmnopqrstuvwxyz0123456789")
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error")
				}
				if principal != nil {
					t.Fatalf("expected no principal")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if principal == nil || principal.UserID == nil || principal.APIKeyID == nil {
				t.Fatalf("expected principal ids")
			}
			if keyRepo.keys[key.KeyID].LastUsedAt == nil {
				t.Fatalf("expected last used timestamp")
			}
		})
	}
}

func TestMiddlewarePopulatesActorContext(t *testing.T) {
	t.Parallel()
	serviceUserID := uuid.New()
	apiKeyID := uuid.New()
	keyRepo := &fakeAPIKeyRepo{keys: map[string]*domain.APIKey{"key123": {Base: domain.Base{ID: apiKeyID}, KeyID: "key123", OwnerUserID: serviceUserID, Name: "ci", HashedKey: testHashAPIKey("liq_key123.abcdefghijklmnopqrstuvwxyz0123456789")}}}
	identity := &fakeIdentity{serviceAccount: &domain.User{Base: domain.Base{ID: serviceUserID}, Email: "svc@example.com", Role: domain.RoleLicenseManager, IsServiceAccount: true, Active: true}}
	manager := &Manager{identity: identity, users: fakeUserRepo{}, keys: keyRepo, logger: zap.NewNop()}

	_, api := humatest.New(t, huma.DefaultConfig("test", "test"))
	group := huma.NewGroup(api, "/api/v1")
	group.UseMiddleware(manager.Middleware())

	var gotRequestContext app.RequestContext
	var gotPrincipal Principal
	huma.Get(group, "/secure", func(ctx context.Context, _ *struct{}) (*struct {
		OK bool `json:"ok"`
	}, error) {
		gotRequestContext = app.RequestContextFromContext(ctx)
		principal, ok := PrincipalFromContext(ctx)
		if !ok {
			t.Fatalf("expected principal in context")
		}
		gotPrincipal = principal
		return &struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "secure"
		o.Tags = []string{"test"}
		o.Metadata = map[string]any{"resource": "vendors", "action": "read"}
	})

	resp := api.Get("/api/v1/secure", "X-API-Key: liq_key123.abcdefghijklmnopqrstuvwxyz0123456789", "X-Request-ID: req-123")
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if gotRequestContext.ActorUserID == nil || gotRequestContext.ActorAPIKeyID == nil {
		t.Fatalf("expected actor ids in request context")
	}
	if gotPrincipal.Role != domain.RoleLicenseManager {
		t.Fatalf("unexpected role %s", gotPrincipal.Role)
	}
	if gotRequestContext.SessionID != "req-123" {
		t.Fatalf("expected session id to propagate")
	}
}

func TestOIDCClaimRoleMapping(t *testing.T) {
	t.Parallel()
	authenticator := &OIDCAuthenticator{
		cfg: config.OIDCConfig{Audience: "licenseiq", RoleClaim: "roles", RoleMappings: map[string]domain.Role{"admins": domain.RoleAdministrator}, DefaultRole: domain.RoleViewer},
		verifier: fakeVerifier{token: fakeToken{
			claims: map[string]any{"email": "oidc@example.com", "name": "OIDC User", "roles": "admins", "aud": []string{"licenseiq"}},
			sub:    "subject-123",
			aud:    []string{"licenseiq"},
			exp:    time.Now().Add(time.Hour),
		}},
	}
	user, err := authenticator.Authenticate(context.Background(), "any")
	if err != nil {
		t.Fatalf("expected auth success, got %v", err)
	}
	if user.Role != domain.RoleAdministrator {
		t.Fatalf("expected mapped admin role, got %s", user.Role)
	}
	if user.ExternalSubject != "subject-123" || user.Email != "oidc@example.com" {
		t.Fatalf("unexpected identity mapping: %+v", user)
	}
}

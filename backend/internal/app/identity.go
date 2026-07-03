package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
)

type IdentityService interface {
	ListServiceAccounts(context.Context, domain.ListFilter) ([]domain.User, error)
	GetServiceAccount(context.Context, uuid.UUID) (*domain.User, error)
	CreateServiceAccount(context.Context, domain.User) (*domain.User, error)
	UpdateServiceAccount(context.Context, uuid.UUID, domain.User) (*domain.User, error)
	DeleteServiceAccount(context.Context, uuid.UUID) error
	UpsertAuthenticatedUser(context.Context, domain.User) (*domain.User, error)
	ListAPIKeys(context.Context, uuid.UUID, domain.ListFilter) ([]domain.APIKey, error)
	GetAPIKey(context.Context, uuid.UUID) (*domain.APIKey, error)
	GetAPIKeyByKeyID(context.Context, string) (*domain.APIKey, error)
	CreateAPIKey(context.Context, domain.APIKey) (*domain.APIKey, string, error)
	CreateAPIKeyWithToken(context.Context, domain.APIKey, string) (*domain.APIKey, error)
	DeleteAPIKey(context.Context, uuid.UUID) error
}

type identityService struct {
	users  domain.UserRepository
	keys   domain.APIKeyRepository
	audits domain.AuditRepository
}

func NewIdentityService(users domain.UserRepository, keys domain.APIKeyRepository, audits domain.AuditRepository) IdentityService {
	return &identityService{users: users, keys: keys, audits: audits}
}

func (s *identityService) ListServiceAccounts(ctx context.Context, filter domain.ListFilter) ([]domain.User, error) {
	items, err := s.users.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(items))
	for _, user := range items {
		if user.IsServiceAccount {
			out = append(out, user)
		}
	}
	return out, nil
}

func (s *identityService) GetServiceAccount(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !user.IsServiceAccount {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (s *identityService) CreateServiceAccount(ctx context.Context, input domain.User) (*domain.User, error) {
	input.IsServiceAccount = true
	if !input.Active {
		input.Active = true
	}
	created := input
	if err := s.users.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "user", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *identityService) UpdateServiceAccount(ctx context.Context, id uuid.UUID, input domain.User) (*domain.User, error) {
	previous, err := s.GetServiceAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	input.Base = previous.Base
	input.ID = id
	input.IsServiceAccount = true
	updated := input
	if err := s.users.Update(ctx, &updated); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "user", updated.ID, previous, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *identityService) DeleteServiceAccount(ctx context.Context, id uuid.UUID) error {
	previous, err := s.GetServiceAccount(ctx, id)
	if err != nil {
		return err
	}
	if err := s.users.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "user", id, previous, nil)
}

func (s *identityService) UpsertAuthenticatedUser(ctx context.Context, input domain.User) (*domain.User, error) {
	input.IsServiceAccount = false
	if !input.Active {
		input.Active = true
	}
	if input.ExternalSubject != "" {
		if existing, err := s.users.GetByExternalSubject(ctx, input.ExternalSubject); err == nil {
			input.Base = existing.Base
			input.ID = existing.ID
			updated := input
			if err := s.users.Update(ctx, &updated); err != nil {
				return nil, err
			}
			if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "user", updated.ID, existing, updated); err != nil {
				return nil, err
			}
			return &updated, nil
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	if input.Email != "" {
		if existing, err := s.users.GetByEmail(ctx, input.Email); err == nil {
			input.Base = existing.Base
			input.ID = existing.ID
			updated := input
			if err := s.users.Update(ctx, &updated); err != nil {
				return nil, err
			}
			if err := writeAudit(ctx, s.audits, domain.AuditActionUpdate, "user", updated.ID, existing, updated); err != nil {
				return nil, err
			}
			return &updated, nil
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	created := input
	if err := s.users.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "user", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *identityService) ListAPIKeys(ctx context.Context, ownerID uuid.UUID, filter domain.ListFilter) ([]domain.APIKey, error) {
	items, err := s.keys.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]domain.APIKey, 0, len(items))
	for _, key := range items {
		if key.OwnerUserID == ownerID {
			out = append(out, key)
		}
	}
	return out, nil
}

func (s *identityService) GetAPIKey(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	return s.keys.Get(ctx, id)
}

func (s *identityService) GetAPIKeyByKeyID(ctx context.Context, keyID string) (*domain.APIKey, error) {
	return s.keys.GetByKeyID(ctx, keyID)
}

func (s *identityService) CreateAPIKey(ctx context.Context, input domain.APIKey) (*domain.APIKey, string, error) {
	input.KeyID = generateKeyID()
	input.Active = true
	plain, err := generateAPIKey(input.KeyID)
	if err != nil {
		return nil, "", err
	}
	created, err := s.CreateAPIKeyWithToken(ctx, input, plain)
	return created, plain, err
}

func (s *identityService) CreateAPIKeyWithToken(ctx context.Context, input domain.APIKey, token string) (*domain.APIKey, error) {
	keyID, _, ok := APIKeyTokenParts(token)
	if !ok {
		return nil, domain.ErrValidation
	}
	user, err := s.users.Get(ctx, input.OwnerUserID)
	if err != nil {
		return nil, err
	}
	if !user.IsServiceAccount || !user.Active {
		return nil, domain.ErrConflict
	}
	input.KeyID = keyID
	input.Active = true
	input.HashedKey = hashAPIKey(token)
	created := input
	if err := s.keys.Create(ctx, &created); err != nil {
		return nil, err
	}
	if err := writeAudit(ctx, s.audits, domain.AuditActionCreate, "api_key", created.ID, nil, created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *identityService) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	previous, err := s.keys.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.keys.Delete(ctx, id); err != nil {
		return err
	}
	return writeAudit(ctx, s.audits, domain.AuditActionDelete, "api_key", id, previous, nil)
}

func generateKeyID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
}

func generateAPIKey(keyID string) (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return "liq_" + keyID + "." + base64.RawURLEncoding.EncodeToString(secret), nil
}

func hashAPIKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func APIKeyPrefix(token string) (string, bool) {
	if !strings.HasPrefix(token, "liq_") {
		return "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(token, "liq_"), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0], true
}

func APIKeyHashMatches(expectedHash, token string) bool {
	computed := hashAPIKey(token)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(computed)) == 1
}

func APIKeyTokenParts(token string) (string, string, bool) {
	if !strings.HasPrefix(token, "liq_") {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(token, "liq_"), ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

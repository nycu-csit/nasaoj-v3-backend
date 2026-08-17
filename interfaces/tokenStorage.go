package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// TokenStorage persists static bearer tokens. Postgres is the authoritative
// store; when Redis is configured, lookups are cached in front of it.
type TokenStorage interface {
	GetTokens(
		ctx context.Context,
	) ([]models.Token, error)

	// GetToken returns the stored token record (Privs, UserId, IsEnabled) so
	// callers can decide whether to trust Privs as-is (static tokens) or
	// resolve live privileges from UserId (user tokens).
	GetToken(
		ctx context.Context,
		token string,
	) (models.Token, error)

	SetAdminToken(
		ctx context.Context,
		token string,
	) error

	// GenerateTokenWithoutExpire issues a token with no expiry. userId is 0
	// for static (admin/worker) tokens, or the owning user's id for a token
	// whose privileges/namespaces should be resolved live on each request.
	// description is required and shown by `judge admin get tokens`.
	GenerateTokenWithoutExpire(
		ctx context.Context,
		privs models.Privileges,
		userId int64,
		description string,
	) (string, error)

	DisableToken(
		ctx context.Context,
		token string,
	) error

	EnableToken(
		ctx context.Context,
		token string,
	) error
}

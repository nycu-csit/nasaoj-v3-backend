package interfaces

import (
	"context"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type TokenStorage interface {
	GetTokens(
		ctx context.Context,
	) (map[string]models.Token, error)

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

	// GenerateToken issues a token that expires after duration. userId is 0
	// for static (admin/worker) tokens, or the owning user's id for a token
	// whose privileges/namespaces should be resolved live on each request.
	// description is required and shown by `judge admin get tokens`.
	GenerateToken(
		ctx context.Context,
		privs models.Privileges,
		userId int64,
		description string,
		duration time.Duration,
	) (string, error)

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

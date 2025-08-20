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

	GetPrivs(
		ctx context.Context,
		token string,
	) (models.Privileges, error)

	GenerateToken(
		ctx context.Context,
		privs models.Privileges,
		duration time.Duration,
	) (string, error)

	GenerateTokenWithoutExpire(
		ctx context.Context,
		privs models.Privileges,
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

package workflows

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

var RegisterUserHooks []func(*models.UserResource) error

func RegisterUser(
	ctx context.Context,
	db interfaces.Database,
	csId string,
) error {
	user := models.UserResource{
		Username: csId,
		Source:   models.LoginSourceCSSSO,
		SourceId: csId,
	}

	for _, hook := range RegisterUserHooks {
		if err := hook(&user); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to register user hook",
					zap.Error(err))
			return err
		}
	}

	return db.UpsertUser(ctx, user)
}

func init() {
	// Add RegisterUserHooks here
}

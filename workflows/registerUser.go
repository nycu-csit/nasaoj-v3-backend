package workflows

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var RegisterUserHooks []func(*models.UserResource) error

func RegisterUser(
	ctx context.Context,
	db interfaces.Database,
	csId string,
) error {
	user := models.UserResource{
		CsId: csId,
	}

	for _, hook := range RegisterUserHooks {
		if err := hook(&user); err != nil {
			return fmt.Errorf("register user hook failed, %w", err)
		}
	}

	return db.UpsertUser(ctx, user)
}

func init() {
	// Add RegisterUserHooks here
}

package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type Database interface {
	GetUserResource(
		ctx context.Context,
		id int64,
	) (models.UserResource, error)

	GetUserPass(
		ctx context.Context,
		username string,
		passhash string,
	) (models.UserResource, error)

	GetUsers(
		ctx context.Context,
	) ([]models.UserResource, error)

	DeleteUser(
		ctx context.Context,
		id int64,
	) error

	ExistUserResource(
		ctx context.Context,
		id int64,
	) (bool, error)

	ExistUserResourceBySourceId(
		ctx context.Context,
		source models.LoginSourceType,
		sourceId string,
	) (bool, error)

	GetUserResourceBySourceId(
		ctx context.Context,
		source models.LoginSourceType,
		sourceId string,
	) (models.UserResource, error)

	InsertUserResources(
		ctx context.Context,
		users []models.UserResource,
		creds []models.PlainCred,
	) error

	UpsertUser(
		ctx context.Context,
		user models.UserResource,
	) error

	GetSession(
		ctx context.Context,
		id int64,
	) (models.Session, error)

	UpsertSession(
		ctx context.Context,
		session *models.Session,
	) error

	UpdateRefreshToken(
		ctx context.Context,
		id int64,
	) (models.Session, error)

	InvalidateSession(
		ctx context.Context,
		id int64,
	) error
}

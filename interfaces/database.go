package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type Database interface {
	GetUserResource(
		ctx context.Context,
		csId string,
	) (models.UserResource, error)

	GetUsers(
		ctx context.Context,
	) ([]models.UserResource, error)

	DeleteUser(
		ctx context.Context,
		csId string,
	) error

	ExistUserResource(
		ctx context.Context,
		csId string,
	) (bool, error)

	InsertUserResources(
		ctx context.Context,
		userResources []models.UserResource,
	) error

	UpsertUser(
		ctx context.Context,
		user models.UserResource,
	) error

	GetSession(
		ctx context.Context,
		csId string,
	) (models.Session, error)

	UpsertSession(
		ctx context.Context,
		session *models.Session,
	) error

	UpdateRefreshToken(
		ctx context.Context,
		csId string,
	) (models.Session, error)

	InvalidateSession(
		ctx context.Context,
		csId string,
	) error
}

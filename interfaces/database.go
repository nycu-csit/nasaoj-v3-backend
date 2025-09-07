package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type Database interface {
	GetLastSubmission(
		ctx context.Context,
		csId string,
		refId int64,
		subType models.RefType,
	) (models.Submission, error)

	GetSubmission(
		ctx context.Context,
		subId int64,
	) (models.Submission, error)

	GetScores(
		ctx context.Context,
		refId int64,
	) ([]models.UserScore, error)

	GetScoresWithStages(
		ctx context.Context,
		refId int64,
	) ([]models.UserScoreWithStages, error)

	GetUserResource(
		ctx context.Context,
		csId string,
	) (models.UserResource, error)

	GetSubmissionsAndCount(
		ctx context.Context,
		csId string,
		isCsId bool,
		pageNum int,
		isPageNum bool,
		pageSize int,
		refId int64,
		isRefId bool,
		refType models.RefType,
		isRefType bool,
	) ([]models.Submission, int, error)

	DeleteSubmission(
		ctx context.Context,
		subId int64,
	) error

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

	InsertSubmissionByProblem(
		ctx context.Context,
		submission *models.Submission,
		problem models.Problem,
	) error

	InsertSubmissionByTool(
		ctx context.Context,
		submission *models.Submission,
		tool models.Tool,
	) error

	UpsertUser(
		ctx context.Context,
		user models.UserResource,
	) error

	SkipJobs(
		ctx context.Context,
		subId int64,
	) error

	UpdateJob(
		ctx context.Context,
		job *models.Job,
	) error

	IncreaseScore(
		ctx context.Context,
		subId int64,
		stageName string,
		delta int64,
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

	DeleteUserAndSubmissions(
		ctx context.Context,
		csId string,
	) error
}

package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type Database interface {
	// Get Operations
	GetLastSubmission(
		ctx context.Context,
		csId string,
		problemId models.ProblemIdType,
	) (models.Submission, error)

	GetSubmission(
		ctx context.Context,
		subId int64,
	) (models.Submission, error)

	GetScores(
		ctx context.Context,
		problemId int64,
	) ([]models.UserScore, error)

	GetScoresWithStages(
		ctx context.Context,
		problemId int64,
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
		problemId int64,
		isProblemId bool,
	) ([]models.Submission, int, error)

	GetUsers(
		ctx context.Context,
	) ([]models.UserResource, error)

	// Exist Operation
	ExistUserResource(
		ctx context.Context,
		csId string,
	) (bool, error)

	// Insert Operation
	InsertUserResources(
		ctx context.Context,
		userResources []models.UserResource,
	) error

	InsertSubmissionAndJobs(
		ctx context.Context,
		submission *models.Submission,
		problem models.Problem,
	) error

	InsertNewUserFromResources(
		ctx context.Context,
		csId string,
	) error

	// Update Operation
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
}

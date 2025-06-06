package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type Database interface {
	GetLastSubmission(
		ctx context.Context,
		csId string,
		problemId models.ProblemIdType,
	) (models.Submission, error)

	GetSubmission(
		ctx context.Context,
		subId int64,
	) (models.Submission, error)

	SkipJobs(
		ctx context.Context,
		subId int64,
	) error

	InsertSubmissionAndJobs(
		ctx context.Context,
		submission *models.Submission,
		problem models.Problem,
	) error

	UpdateJob(
		ctx context.Context,
		job *models.Job,
	) error

	GetUserResource(
		ctx context.Context,
		csId string,
	) (models.UserResource, error)

	IncreaseScore(
		ctx context.Context,
		subId int64,
		delta int64,
	) error
}

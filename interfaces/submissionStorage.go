package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type SubmissionStorage interface {
	GetSubmission(
		ctx context.Context,
		subId int64,
	) (models.Submission, error)

	GetLastSubmission(
		ctx context.Context,
		id int64,
		refId int64,
		subType models.RefType,
	) (models.Submission, error)

	GetSubmissionsAndCount(
		ctx context.Context,
		id int64,
		isOwnerId bool,
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

	DeleteUserAndSubmissions(
		ctx context.Context,
		id int64,
	) error

	SkipJobs(
		ctx context.Context,
		subId int64,
	) error

	UpdateJob(
		ctx context.Context,
		job *models.Job,
	) error

	CancelSubmission(
		ctx context.Context,
		subId int64,
		ownerId int64,
	) error
}

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
		subType int,
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

	InsertProblemSubmission(
		ctx context.Context,
		submission *models.Submission,
		problem models.Problem,
	) error

	InsertToolSubmission(
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

	GetProblem(
		ctx context.Context,
		sha1 string,
	) (models.Problem, error)

	GetProblemById(
		ctx context.Context,
		problemId int64,
	) (models.Problem, error)

	HydrateProblem(
		ctx context.Context,
		problem *models.Problem,
	) error

	HydrateTool(
		ctx context.Context,
		tool *models.Tool,
	) error

	GetProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	GetProblemScores(
		ctx context.Context,
		sha1s []string,
	) (map[string]int64, error)

	GetAllProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	InsertProblem(
		ctx context.Context,
		problem models.Problem,
	) error

	DeleteProblemAndSubmissions(
		ctx context.Context,
		sha1 string,
	) error

	InsertContentFiles(
		ctx context.Context,
		contentFiles []models.ContentFile,
	) error

	GetContentFiles(
		ctx context.Context,
		sha1s []string,
	) ([]models.ContentFile, error)

	GetAllContentFiles(
		ctx context.Context,
	) ([]models.ContentFile, error)

	// Update state of is_enabled
	UpdateProblem(
		ctx context.Context,
		sha1 string,
		enable bool,
	) error

	// For testing purpose
	EnableAllProblem(
		ctx context.Context,
	) error

	GetTool(
		ctx context.Context,
		sha1 string,
	) (models.Tool, error)

	GetToolById(
		ctx context.Context,
		problemId int64,
	) (models.Tool, error)

	GetTools(
		ctx context.Context,
	) (map[string]models.Tool, error)

	GetAllTools(
		ctx context.Context,
	) (map[string]models.Tool, error)

	InsertTool(
		ctx context.Context,
		tool models.Tool,
	) error

	UpdateTool(
		ctx context.Context,
		sha1 string,
		enable bool,
	) error

	DeleteToolAndSubmissions(
		ctx context.Context,
		sha1 string,
	) error

	DeleteUserAndSubmissions(
		ctx context.Context,
		csId string,
	) error
}

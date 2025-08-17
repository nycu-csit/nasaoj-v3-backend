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

	UpsertUser(
		ctx context.Context,
		user models.UserResource,
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

	GetProblem(
		ctx context.Context,
		sha1 string,
	) (models.Problem, error)

	GetProblemById(
		ctx context.Context,
		problemId models.ProblemIdType,
	) (models.Problem, error)

	GetProblemOrToolById(
		ctx context.Context,
		problemId models.ProblemIdType,
	) (models.Problem, error)

	GetProblemOrTool(
		ctx context.Context,
		sha1 string,
	) (models.Problem, error)

	HydrateProblem(
		ctx context.Context,
		problm *models.Problem,
	) error

	GetProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

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
		problemId models.ProblemIdType,
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
}

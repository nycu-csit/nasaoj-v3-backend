package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
)

type AdminSubmitInput struct {
	SubmitInput `json:",inline"`
	AsCsId      string `json:"asCsId"`
}

type SubmitInput struct {
	// Submit problem ID
	RefSHA1   string            `json:"refSHA1"`
	Type      int               `json:"type"`
	InputArgs map[string]string `json:"inputArgs"`
}

type SubmitOutput struct {
	Id        int64             `json:"id"`
	RefSHA1   string            `json:"refSHA1"`
	RefId     int64             `json:"refId"`
	InputArgs map[string]string `json:"inputArgs"`

	CsId      string `json:"csId"`
	CreatedAt int64  `json:"createdAt"`
}

// PostAdminSubmit
//
//	@Param		request	body	AdminSubmitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	SubmitOutput
//	@Failure	400
//	@Failure	401
//	@Failure	429
//
//	@Router		/api/admin/submit [post]
func putAdminSubmitPlaceholder() {}

// PostSubmit
//
//	@Param		request	body	SubmitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	SubmitOutput
//	@Failure	400
//	@Failure	401
//	@Failure	429
//
//	@Router		/api/submit [post]
func PostSubmit(
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
	tools interfaces.ToolStorage,
	tokens interfaces.TokenStorage,

	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		var input SubmitInput
		var csId string
		if isAdmin {
			var adminInput AdminSubmitInput
			if err := json.Unmarshal(b, &adminInput); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "failed to unmarshal json",
					OriginError: err,
				}
			}
			input = adminInput.SubmitInput
			csId = adminInput.AsCsId
		} else {
			if err := json.Unmarshal(b, &input); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "failed to unmarshal json",
					OriginError: err,
				}
			}
			privs, ok := req.Context().Value(
				types.Privileges("")).(models.Privileges)
			if !ok {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "privs not exist",
					OriginError: err,
				}
			}

			csId = privs.OwnerCsId
		}

		resource, err := db.GetUserResource(ctx, csId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "resource query failed or user not exist",
				OriginError: err,
			}
		}

		var submission models.Submission
		switch input.Type {
		case models.RefTypeProblem:
			problem, err := problems.GetProblem(ctx, input.RefSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "problem not exist",
					OriginError: err,
				}
			}

			if !isAdmin && !problem.IsEnabled {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "problem not enabled",
				}
			}

			if !isAdmin {
				// Check last submission
				lastSubmit, err := db.GetLastSubmission(ctx,
					resource.CsId, problem.ProblemId, models.RefTypeProblem)
				if err == nil {
					// Check Job status
					if slices.ContainsFunc(lastSubmit.Jobs, func(job *models.Job) bool {
						return job.Status == models.StatusPending ||
							job.Status == models.StatusRunning
					}) {
						availableAt := time.Unix(lastSubmit.CreatedAt, 0).
							Add(time.Duration(problem.Timeout) * time.Second)
						if time.Now().Unix() < availableAt.Unix() {
							w.Header().Set("Retry-After", fmt.Sprintf("%d",
								int64(availableAt.Sub(time.Now()).Seconds())))
							return middlewares.HTTPError{
								StatusCode: http.StatusTooManyRequests,
								Message:    "submit in cooldown",
							}
						}
					}
				} else if err != sql.ErrNoRows {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "failed to query last submission",
						OriginError: err,
					}
				}
			}

			submission = models.Submission{
				ProblemSHA1: input.RefSHA1,
				ProblemId:   problem.ProblemId,
				Type:        models.RefTypeProblem,

				CsId:      resource.CsId,
				CreatedAt: time.Now().Unix(),
				Args:      make(map[string]string),

				Score: models.Score{
					Score:  0,
					Stages: make(map[string]int64),
				},
			}

			for key, val := range resource.Map() {
				submission.Args[key] = val
			}

			if err := db.InsertProblemSubmission(
				ctx, &submission, problem); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to insert problem submission",
					OriginError: err,
				}
			}

			if err := workflows.RunProblem(
				ctx, jobQue, db, s3, problems, tokens, submission); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to run problem",
					OriginError: err,
				}
			}

		case models.RefTypeTool:
			tool, err := tools.GetTool(ctx, input.RefSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "tool not exist",
					OriginError: err,
				}
			}

			if !isAdmin && !tool.IsEnabled {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "tool not enabled",
				}
			}

			if !isAdmin {
				// Check last submission
				lastSubmit, err := db.GetLastSubmission(ctx,
					resource.CsId, tool.ToolId, models.RefTypeTool)
				if err == nil {
					// Check Job status
					if slices.ContainsFunc(lastSubmit.Jobs, func(job *models.Job) bool {
						return job.Status == models.StatusPending ||
							job.Status == models.StatusRunning
					}) {
						availableAt := time.Unix(lastSubmit.CreatedAt, 0).
							Add(time.Duration(tool.Timeout) * time.Second)
						if time.Now().Unix() < availableAt.Unix() {
							w.Header().Set("Retry-After", fmt.Sprintf("%d",
								int64(availableAt.Sub(time.Now()).Seconds())))
							return middlewares.HTTPError{
								StatusCode: http.StatusTooManyRequests,
								Message:    "submit in cooldown",
							}
						}
					}
				} else if err != sql.ErrNoRows {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "failed to query last submission",
						OriginError: err,
					}
				}
			}

			submission = models.Submission{
				ToolSHA1: input.RefSHA1,
				ToolId:   tool.ToolId,
				Type:     models.RefTypeTool,

				CsId:      resource.CsId,
				CreatedAt: time.Now().Unix(),
				Args:      make(map[string]string),
				InputArgs: make(map[string]string),

				Score: models.Score{
					Score:  0,
					Stages: make(map[string]int64),
				},
			}

			for _, key := range tool.InputArgs {
				submission.InputArgs[key] = input.InputArgs[key]
			}

			for key, val := range resource.Map() {
				submission.Args[key] = val
			}

			if err := db.InsertToolSubmission(ctx, &submission, tool); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to insert tool submission",
					OriginError: err,
				}
			}

			if err := workflows.RunTool(
				ctx, jobQue, db, s3, tools, tokens, submission); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to run tool",
					OriginError: err,
				}
			}
		}

		switch submission.Type {
		case models.RefTypeProblem:
			return bunrouter.JSON(w, SubmitOutput{
				Id:        submission.Id,
				RefSHA1:   submission.ProblemSHA1,
				RefId:     submission.ProblemId,
				InputArgs: submission.InputArgs,

				CsId:      submission.CsId,
				CreatedAt: submission.CreatedAt,
			})
		case models.RefTypeTool:
			return bunrouter.JSON(w, SubmitOutput{
				Id:        submission.Id,
				RefSHA1:   submission.ToolSHA1,
				RefId:     submission.ToolId,
				InputArgs: submission.InputArgs,

				CsId:      submission.CsId,
				CreatedAt: submission.CreatedAt,
			})
		default:
			return middlewares.HTTPError{
				StatusCode: http.StatusInternalServerError,
				Message:    "unhandled case",
			}
		}
	}
}

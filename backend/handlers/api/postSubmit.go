package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/workflows"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type AdminSubmitInput struct {
	SubmitInput `json:",inline"`
	AsOwnerId   int64 `json:"asOwnerId"`
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

	OwnerId   int64 `json:"ownerId"`
	CreatedAt int64 `json:"createdAt"`
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
func (self *Handlers) PostSubmit(
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
		var ownerId int64
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
			ownerId = adminInput.AsOwnerId
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

			ownerId = privs.OwnerId
		}

		resource, err := self.db.GetUserResource(ctx, ownerId)
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
			problem, err := self.problems.GetProblem(ctx, input.RefSHA1)
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
				lastSubmit, err := self.submissions.GetLastSubmission(ctx,
					resource.Id, problem.ProblemId, models.RefTypeProblem)
				if err == nil {
					// Check Job status
					if slices.ContainsFunc(lastSubmit.Jobs, func(job *models.Job) bool {
						return !job.IsDone()
					}) {
						return middlewares.HTTPError{
							StatusCode: http.StatusPreconditionFailed,
							Message:    "wait for previous request to finish",
						}
					}

					availableAt := time.Unix(lastSubmit.CreatedAt, 0).
						Add(time.Duration(problem.Cooldown) * time.Second)
					if time.Now().Unix() < availableAt.Unix() {
						w.Header().Set("Retry-After", fmt.Sprintf("%d",
							int64(availableAt.Sub(time.Now()).Seconds())))
						return middlewares.HTTPError{
							StatusCode: http.StatusTooManyRequests,
							Message:    "submit in cooldown",
						}
					}
				} else if !errors.Is(err, sql.ErrNoRows) {
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

				OwnerId:   resource.Id,
				CreatedAt: time.Now().Unix(),
				Args:      make(map[string]string),
			}

			for key, val := range resource.Map() {
				submission.Args[key] = val
			}

			if err := self.submissions.InsertSubmissionByProblem(
				ctx, &submission, problem); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to insert problem submission",
					OriginError: err,
				}
			}

			if err := workflows.RunProblem(ctx, self.jobQue, self.db, self.s3,
				self.problems, self.contents, self.jobs,
				self.tokens, submission); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to run problem",
					OriginError: err,
				}
			}

		case models.RefTypeTool:
			tool, err := self.tools.GetTool(ctx, input.RefSHA1)
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
				lastSubmit, err := self.submissions.GetLastSubmission(ctx,
					resource.Id, tool.ToolId, models.RefTypeTool)
				if err == nil {
					// Check Job status
					if slices.ContainsFunc(lastSubmit.Jobs, func(job *models.Job) bool {
						return !job.IsDone()
					}) {
						return middlewares.HTTPError{
							StatusCode: http.StatusPreconditionFailed,
							Message:    "wait for previous request to finish",
						}
					}
					availableAt := time.Unix(lastSubmit.CreatedAt, 0).
						Add(time.Duration(tool.Cooldown) * time.Second)
					if time.Now().Unix() < availableAt.Unix() {
						w.Header().Set("Retry-After", fmt.Sprintf("%d",
							int64(availableAt.Sub(time.Now()).Seconds())))
						return middlewares.HTTPError{
							StatusCode: http.StatusTooManyRequests,
							Message:    "submit in cooldown",
						}
					}
				} else if !errors.Is(err, sql.ErrNoRows) {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "failed to query last submission",
						OriginError: err,
					}
				}
			}

			if err := utils.ValidateInputArgs(tool.InputArgs, input.InputArgs); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     err.Error(),
					OriginError: err,
				}
			}

			submission = models.Submission{
				ToolSHA1: input.RefSHA1,
				ToolId:   tool.ToolId,
				Type:     models.RefTypeTool,

				OwnerId:   resource.Id,
				CreatedAt: time.Now().Unix(),
				Args:      make(map[string]string),
				InputArgs: make(map[string]string),
			}

			for _, arg := range tool.InputArgs {
				submission.InputArgs[arg.Name] = input.InputArgs[arg.Name]
			}

			for key, val := range resource.Map() {
				submission.Args[key] = val
			}

			if err := self.submissions.InsertSubmissionByTool(
				ctx, &submission, tool); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to insert tool submission",
					OriginError: err,
				}
			}

			if err := workflows.RunTool(ctx, self.jobQue, self.db, self.s3,
				self.tools, self.contents, self.jobs, self.tokens,
				submission); err != nil {
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

				OwnerId:   submission.OwnerId,
				CreatedAt: submission.CreatedAt,
			})
		case models.RefTypeTool:
			return bunrouter.JSON(w, SubmitOutput{
				Id:        submission.Id,
				RefSHA1:   submission.ToolSHA1,
				RefId:     submission.ToolId,
				InputArgs: submission.InputArgs,

				OwnerId:   submission.OwnerId,
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

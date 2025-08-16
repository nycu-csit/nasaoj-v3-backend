package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
)

type adminSubmitInput struct {
	submitInput `json:",inline"`
	AsCsId      string `json:"asCsId"`
}

type submitInput struct {
	// Submit problem ID
	ProblemSHA1 string `json:"problemSHA1"`
}

type submitOutput struct {
	Id          int64  `json:"id"`
	ProblemSHA1 string `json:"problemSHA1"`
	ProblemId   int64  `json:"problemId"`
	ProblemName string `json:"problemName"`

	CsId     string `json:"csId"`
	SubmitAt int64  `json:"submitAt"`
}

// PostAdminSubmit
//
//	@Param		request	body	adminSubmitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400
//	@Failure	401
//	@Failure	429
//
//	@Router		/api/admin/submit [post]
func putAdminSubmitPlaceholder() {}

// PostSubmit
//
//	@Param		request	body	submitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400	{object}	string
//	@Failure	401	{object}	string
//	@Failure	429	{object}	string
//
//	@Router		/api/submit [post]
func PostSubmit(
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
	tokens interfaces.TokenStorage,

	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		if isAdmin {
			fmt.Fprintf(os.Stderr, "POST /api/admin/submit\n")
		} else {
			fmt.Fprintf(os.Stderr, "POST /api/submit\n")
		}

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		var problemSHA1 string
		var problem models.Problem
		var resource models.UserResource
		if isAdmin {
			var input adminSubmitInput
			if err := json.Unmarshal(b, &input); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "json unmarshal failed",
					OriginError: err,
				}
			}

			problemSHA1 = input.ProblemSHA1
			problem, err = problems.GetProblem(ctx, problemSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "get problem failed",
					OriginError: err,
				}
			}

			resource, err = db.GetUserResource(ctx, input.AsCsId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "resource query failed or user not exist",
					OriginError: err,
				}
			}
		} else {
			var input submitInput
			if err := json.Unmarshal(b, &input); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "json unmarshal failed",
					OriginError: err,
				}
			}

			problemSHA1 = input.ProblemSHA1
			problem, err = problems.GetProblemOrTool(ctx, problemSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "get problem failed",
					OriginError: err,
				}
			}

			claim, ok := req.Context().Value(
				types.AccessToken("")).(models.AccessTokenClaim)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "access token missing",
				}
			}
			resource = claim.Resource
		}

		if !problem.IsEnabled {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "problem not enabled",
			}
		}

		// Check last submission
		lastSubmit, err := db.GetLastSubmission(ctx,
			resource.CsId, problem.ProblemId)
		if err == nil {
			// Check Job status
			if slices.ContainsFunc(lastSubmit.Jobs, func(job *models.Job) bool {
				return job.Status == models.StatusPending ||
					job.Status == models.StatusRunning
			}) {
				availableAt := time.Unix(lastSubmit.SubmitAt, 0).
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
		}

		submission := models.Submission{
			ProblemSHA1: problemSHA1,
			ProblemName: problem.Name,
			ProblemId:   problem.ProblemId,

			CsId:     resource.CsId,
			SubmitAt: time.Now().Unix(),
			Args:     make(map[string]string),

			Score: models.Score{
				Score:  0,
				Stages: make(map[string]int64),
			},
		}

		for key, val := range resource.Map() {
			submission.Args[key] = val
		}

		if err = db.InsertSubmissionAndJobs(ctx, &submission, problem); err != nil {
			return err
		}

		if err = workflows.RunProblem(
			ctx, jobQue, db, s3, problems, tokens, submission,
		); err != nil {
			return err
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, submitOutput{
			Id:          submission.Id,
			ProblemSHA1: problemSHA1,
			ProblemId:   submission.ProblemId,
			ProblemName: submission.ProblemName,

			CsId:     submission.CsId,
			SubmitAt: submission.SubmitAt,
		})
		return nil
	}
}

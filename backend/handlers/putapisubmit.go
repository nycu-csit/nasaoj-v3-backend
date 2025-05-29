package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"text/template"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
)

type adminSubmitInput struct {
	submitInput `json:",inline"`
	AsCsId      string `json:"asCsId"`
}

type submitInput struct {
	// Submit problem ID
	ProblemId int64 `json:"problemId"`
}

type submitOutput struct {
	Id          int64  `json:"id"`
	ProblemId   int64  `json:"problemId"`
	ProblemName string `json:"problemName"`

	CsId     string `json:"csId"`
	SubmitAt int64  `json:"submitAt"`
}

// PutAdminSubmit
//
//	@Param		request	body	adminSubmitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400
//	@Failure	401
//	@Failure	429
//
//	@Router		/api/admin/submit [put]
func putAdminSubmitPlaceholder() {}

// PutSubmit
//
//	@Param		request	body	submitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400	{object}	string
//	@Failure	401	{object}	string
//	@Failure	429	{object}	string
//
//	@Router		/api/submit [put]
func PutSubmit(
	ctx context.Context,
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,

	problems utils.ProblemRepository,
	testcases map[string]*template.Template,

	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		if isAdmin {
			fmt.Fprintf(os.Stderr, "PUT /api/admin/submit\n")
		} else {
			fmt.Fprintf(os.Stderr, "PUT /api/submit\n")
		}

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

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

			var ok bool
			problem, ok = problems[input.ProblemId]
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "problem not exist",
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

			var ok bool
			problem, ok = problems[input.ProblemId]
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "problem not exist",
				}
			}

			resource, ok = req.Context().Value(
				middlewares.UserResourceType("")).(models.UserResource)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "user resource missing",
				}
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
			ProblemName: problem.Name,
			ProblemId:   problem.ProblemId,

			CsId:     resource.CsId,
			SubmitAt: time.Now().Unix(),
			Args:     make(map[string]string),

			Score: models.UserScore{
				Score:  0,
				Stages: make(map[string]int64),
			},
		}

		for key, val := range resource.Map() {
			submission.Args[key] = val
		}

		err = db.InsertSubmissionAndJobs(ctx, &submission, problem)
		if err != nil {
			return err
		}

		err = workflows.RunProblem(ctx, jobQue, db, s3,
			problems, testcases, submission)
		if err != nil {
			return err
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, submitOutput{
			Id:          submission.Id,
			ProblemId:   submission.ProblemId,
			ProblemName: submission.ProblemName,

			CsId:     submission.CsId,
			SubmitAt: submission.SubmitAt,
		})
		return nil
	}
}

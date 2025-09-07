package implements

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

type BunDatabase struct {
	db *bun.DB
}

func NewBunDatabase(db *bun.DB) *BunDatabase {
	return &BunDatabase{db: db}
}

func (self *BunDatabase) submissionQueryHook(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Job)(nil)).
		Set("status = ?", models.StatusTimeout).
		Where("sub_id = ?", subId).
		Where("deadline < ?", time.Now().Unix()).
		Where("status = ?", models.StatusRunning).
		Exec(ctx)
	return err
}

func (self *BunDatabase) GetLastSubmission(
	ctx context.Context,
	csId string,
	refId int64,
	subType int,
) (models.Submission, error) {
	var ret models.Submission
	query := self.db.NewSelect().
		Model(&ret).
		Where("cs_id = ?", csId).
		Where("type = ?", subType).
		Order("id DESC").
		Limit(1)

	switch subType {
	case models.RefTypeProblem:
		query = query.Where("problem_id = ?", refId)
	case models.RefTypeTool:
		query = query.Where("tool_id = ?", refId)
	}

	err := query.Scan(ctx)
	if err != nil {
		return models.Submission{}, err
	}
	if err = self.submissionQueryHook(ctx, ret.Id); err != nil {
		return models.Submission{}, err
	}
	err = self.db.NewSelect().
		Model(&ret).
		WherePK().
		Relation("Jobs").
		Scan(ctx)
	if err != nil {
		return models.Submission{}, err
	}
	return ret, nil
}

func (self *BunDatabase) GetSubmission(
	ctx context.Context,
	subId int64,
) (models.Submission, error) {
	ret := models.Submission{Id: subId}
	if err := self.submissionQueryHook(ctx, subId); err != nil {
		return models.Submission{}, err
	}

	err := self.db.NewSelect().
		Model(&ret).
		WherePK().
		Relation("Jobs").
		Scan(ctx)
	return ret, err
}

func (self *BunDatabase) SkipJobs(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Job)(nil)).
		Set("status = ?", models.StatusSkip).
		Where("sub_id = ?", subId).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Where("status = ?", models.StatusBlocked).
				WhereOr("status = ?", models.StatusPending)
		}).
		Exec(ctx)
	return err
}

func (self *BunDatabase) UpdateJob(
	ctx context.Context,
	job *models.Job,
) error {
	_, err := self.db.NewUpdate().Model(job).WherePK().Exec(ctx)
	return err
}

func (self *BunDatabase) InsertSubmissionByProblem(
	ctx context.Context,
	submission *models.Submission,
	problem models.Problem,
) error {
	return self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewInsert().
			Model(submission).
			Scan(ctx)
		if err != nil {
			return err
		}

		for stageName, stage := range problem.Stages {
			for _, testcase := range stage.Testcases {
				job := models.Job{
					SubId:     submission.Id,
					StageName: stageName,
					TestName:  testcase.Name,

					Status: models.StatusBlocked,
				}

				_, err = tx.NewInsert().
					Model(&job).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
		}

		_, err = tx.NewUpdate().
			Model(submission).
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (self *BunDatabase) InsertSubmissionByTool(
	ctx context.Context,
	submission *models.Submission,
	tool models.Tool,
) error {
	return self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewInsert().
			Model(submission).
			Scan(ctx)
		if err != nil {
			return err
		}

		job := models.Job{
			SubId:     submission.Id,
			StageName: "tool",
			TestName:  tool.Testcase.Name,

			Status: models.StatusBlocked,
		}
		_, err = tx.NewInsert().
			Model(&job).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (self *BunDatabase) GetUserResource(
	ctx context.Context,
	csId string,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().Model(&ret).
		Where("cs_id = ?", csId).Scan(ctx)
	return ret, err
}

func (self *BunDatabase) IncreaseScore(
	ctx context.Context,
	subId int64,
	stageName string,
	delta int64,
) error {
	return self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		submission := models.Submission{Id: subId}
		err := self.db.NewSelect().
			Column("id", "score_score", "score_stages").
			Model(&submission).
			WherePK().
			Scan(ctx)
		if err != nil {
			return err
		}

		submission.Score.Score += delta
		submission.Score.Stages[stageName] = delta
		_, err = self.db.NewUpdate().
			Column("score_score", "score_stages").
			Model(&submission).
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (self *BunDatabase) GetScoresWithStages(
	ctx context.Context,
	refId int64,
) ([]models.UserScoreWithStages, error) {
	ret := []models.UserScoreWithStages{}
	var submissions []models.Submission
	err := self.db.NewSelect().
		DistinctOn("cs_id").
		Model(&submissions).
		Column("cs_id", "score_score", "score_stages", "problem_id").
		OrderExpr("cs_id, id DESC").
		Where("problem_id = ?", refId).
		Scan(ctx)
	if err != nil {
		return []models.UserScoreWithStages{}, err
	}

	for _, submission := range submissions {
		userScore := models.UserScoreWithStages{
			CsId:  submission.CsId,
			Score: submission.Score,
		}
		ret = append(ret, userScore)
	}

	return ret, nil
}

func (self *BunDatabase) GetScores(
	ctx context.Context,
	refId int64,
) ([]models.UserScore, error) {
	ret := []models.UserScore{}
	var submissions []models.Submission
	err := self.db.NewSelect().
		DistinctOn("cs_id").
		Model(&submissions).
		Column("cs_id", "score_score", "problem_id").
		Where("problem_id = ?", refId).
		OrderExpr("cs_id, id DESC").
		Scan(ctx)
	if err != nil {
		return []models.UserScore{}, err
	}

	for _, submission := range submissions {
		userScore := models.UserScore{
			CsId:  submission.CsId,
			Score: submission.Score.Score,
		}
		ret = append(ret, userScore)
	}

	return ret, nil
}

func (self *BunDatabase) InsertUserResources(
	ctx context.Context,
	userResources []models.UserResource,
) error {
	for _, newUser := range userResources {
		err := self.db.RunInTx(ctx, nil,
			func(ctx context.Context, tx bun.Tx) error {
				var oldUser models.UserResource
				err := tx.NewSelect().
					Model(&oldUser).
					Where("cs_id = ?", newUser.CsId).
					Scan(ctx)
				if err == sql.ErrNoRows {
					// Insert New User
					_, err := tx.NewInsert().
						Model(&newUser).
						Exec(ctx)
					if err != nil {
						return err
					}
					return nil
				}
				if err != nil {
					return err
				}

				b, err := utils.JSONMerge(ctx, oldUser, newUser)
				if err != nil {
					tracing.Logger.Ctx(ctx).
						Error("failed to merge json",
							zap.Error(err))
					return err
				}

				var mergedUser models.UserResource
				if err := json.Unmarshal(b, &mergedUser); err != nil {
					tracing.Logger.Ctx(ctx).
						Error("failed to unmarshal json to user resource",
							zap.Error(err))
					return err
				}

				_, err = tx.NewUpdate().
					Model((*models.UserResource)(nil)).
					Set("args = ?", mergedUser.Args).
					Where("cs_id = ?", newUser.CsId).
					Exec(ctx)
				if err != nil {
					return err
				}

				return nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *BunDatabase) ExistUserResource(
	ctx context.Context,
	csId string,
) (bool, error) {
	return self.db.NewSelect().
		Model((*models.UserResource)(nil)).
		Where("cs_id = ?", csId).
		Exists(ctx)
}

func (self *BunDatabase) UpsertUser(
	ctx context.Context,
	user models.UserResource,
) error {
	_, err := self.db.NewInsert().
		Model(&user).
		On("CONFLICT (cs_id) DO UPDATE").
		Set("args = EXCLUDED.args").
		Exec(ctx)
	return err
}

func (self *BunDatabase) GetSubmissionsAndCount(
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
) ([]models.Submission, int, error) {
	query := self.db.NewSelect().
		Order("id DESC").
		Limit(pageSize)
	if isCsId {
		query = query.Where("cs_id = ?", csId)
	}
	if isPageNum {
		query = query.Offset(pageNum * pageSize)
	}
	if isRefType {
		query = query.Where("type = ?", refType)
		if isRefId {
			switch refType {
			case models.RefTypeProblem:
				query = query.Where("problem_id = ?", refId)
			case models.RefTypeTool:
				query = query.Where("tool_id = ?", refId)
			}
		}
	}

	var ret []models.Submission
	count, err := query.
		Model(&ret).
		ScanAndCount(ctx)
	return ret, count, err
}

func (self *BunDatabase) GetUsers(
	ctx context.Context,
) ([]models.UserResource, error) {
	var ret []models.UserResource
	err := self.db.NewSelect().Model(&ret).Scan(ctx)
	return ret, err
}

func (self *BunDatabase) GetSession(
	ctx context.Context,
	csId string,
) (models.Session, error) {
	ret := models.Session{CsId: csId}
	err := self.db.NewSelect().Model(&ret).WherePK().Scan(ctx)
	return ret, err
}

func (self *BunDatabase) UpsertSession(
	ctx context.Context,
	session *models.Session,
) error {
	_, err := self.db.NewInsert().Model(session).
		On("CONFLICT (cs_id) DO UPDATE").
		Set("cs_id = EXCLUDED.cs_id").
		Set("groups = EXCLUDED.groups").
		Set("refresh_token = EXCLUDED.refresh_token").
		Set("is_valid = EXCLUDED.is_valid").
		Exec(ctx)
	return err
}

func (self *BunDatabase) UpdateRefreshToken(
	ctx context.Context,
	csId string,
) (models.Session, error) {
	ret := models.Session{
		CsId: csId,
	}
	err := self.db.NewSelect().Model(&ret).WherePK().Scan(ctx)
	if err != nil {
		return models.Session{}, err
	}
	if err := ret.RotateRefreshToken(); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to rotate refresh token",
				zap.Error(err))
		return models.Session{}, err
	}
	err = self.db.NewUpdate().Model((*models.Session)(nil)).
		Where("cs_id = ?", csId).
		Set("refresh_token = ?", ret.RefreshToken).
		Returning("*").Scan(ctx, &ret)
	return ret, nil
}

func (self *BunDatabase) InvalidateSession(
	ctx context.Context,
	csId string,
) error {
	_, err := self.db.NewUpdate().Model((*models.Session)(nil)).
		Where("cs_id = ?", csId).
		Set("is_valid = ?", false).
		Exec(ctx)
	return err
}

func (self *BunDatabase) GetProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	var problemsMarshaled []models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemsMarshaled).
		OrderExpr("problem_id, sha1").
		DistinctOn("problem_id").
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return map[string]models.Problem{}, err
	}

	ret := make(map[string]models.Problem)
	for _, problem := range problemsMarshaled {
		ret[problem.SHA1], err = problem.Unmarshal()
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal problem",
					zap.String("problem.sha1", problem.SHA1),
					zap.Error(err))
			return map[string]models.Problem{}, err
		}
	}
	return ret, nil
}

func (self *BunDatabase) DeleteSubmission(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewDelete().
		Model((*models.Submission)(nil)).
		Where("id = ?", subId).
		Exec(ctx)
	return err
}

func (self *BunDatabase) DeleteUser(
	ctx context.Context,
	csId string,
) error {
	_, err := self.db.NewDelete().
		Model((*models.UserResource)(nil)).
		Where("cs_id = ?", csId).
		Exec(ctx)
	return err
}

func (self *BunDatabase) DeleteUserAndSubmissions(
	ctx context.Context,
	csId string,
) error {
	return self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewDelete().
				Model((*models.UserResource)(nil)).
				Where("cs_id = ?", csId).
				Exec(ctx)
			if err != nil {
				return err
			}
			_, err = tx.NewDelete().
				Model((*models.Submission)(nil)).
				Where("cs_id = ?", csId).
				Exec(ctx)
			if err != nil {
				return err
			}
			return nil
		})
}

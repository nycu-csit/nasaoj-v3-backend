package implements

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
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
	_, err := self.db.NewUpdate().Model((*models.Job)(nil)).
		Set("status = ?", models.StatusTimeout).
		Where("subid = ?", subId).
		Where("deadline < ?", time.Now().Unix()).
		Where("status = ?", models.StatusRunning).Exec(ctx)
	return err
}

func (self *BunDatabase) GetLastSubmission(
	ctx context.Context,
	csId string,
	problemId int64,
) (models.Submission, error) {
	var ret models.Submission
	err := self.db.NewSelect().Model(&ret).
		Where("csid = ?", csId).
		Where("problemid = ?", problemId).
		Order("id DESC").Limit(1).Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("select submission failed, %w", err)
	}
	if err = self.submissionQueryHook(ctx, ret.Id); err != nil {
		return ret, fmt.Errorf("submission query hook failed, %w", err)
	}
	err = self.db.NewSelect().Model(&ret).
		WherePK().Relation("Jobs").Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("select submission failed, %w", err)
	}
	return ret, nil
}

func (self *BunDatabase) GetSubmission(
	ctx context.Context,
	subId int64,
) (models.Submission, error) {
	ret := models.Submission{Id: subId}
	if err := self.submissionQueryHook(ctx, subId); err != nil {
		return ret, fmt.Errorf("submission query hook failed, %w", err)
	}
	err := self.db.NewSelect().Model(&ret).
		WherePK().Relation("Jobs").Scan(ctx)
	return ret, err
}

func (self *BunDatabase) SkipJobs(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Job)(nil)).
		Set("status = ?", models.StatusSkip).
		Where("subid = ?", subId).
		Where("status = ?", models.StatusPending).Exec(ctx)
	return err
}

func (self *BunDatabase) UpdateJob(
	ctx context.Context,
	job *models.Job,
) error {
	_, err := self.db.NewUpdate().Model(job).WherePK().Exec(ctx)
	return err
}

func (self *BunDatabase) InsertSubmissionAndJobs(
	ctx context.Context,
	submission *models.Submission,
	problem models.Problem,
) error {
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewInsert().Model(submission).Scan(ctx)
		if err != nil {
			return fmt.Errorf("submission insertion failed, %w", err)
		}

		for stageName, stage := range problem.Stages {
			for _, testcase := range stage.Testcases {
				job := models.Job{
					SubId:     submission.Id,
					StageName: stageName,
					TestName:  testcase.Name,

					Status: models.StatusPending,
				}

				_, err = tx.NewInsert().Model(&job).Exec(ctx)
				if err != nil {
					return fmt.Errorf("job insertion failed, %w", err)
				}
			}
		}

		_, err = tx.NewUpdate().Model(submission).WherePK().Exec(ctx)
		if err != nil {
			return fmt.Errorf("submission update failed, %w", err)
		}

		return nil
	})
	return err
}

func (self *BunDatabase) GetUserResource(
	ctx context.Context,
	csId string,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().Model(&ret).
		Where("csid = ?", csId).Scan(ctx)
	return ret, err
}

func (self *BunDatabase) IncreaseScore(
	ctx context.Context,
	subId int64,
	stageName string,
	delta int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Submission)(nil)).
		Set("score_score = score_score + ?", delta).
		Where("id = ?", subId).Exec(ctx)
	if err != nil {
		return fmt.Errorf("db score update failed, %w", err)
	}
	_, err = self.db.NewUpdate().Model((*models.Submission)(nil)).
		Set("score_stages = score_stages || ?",
			map[string]int64{stageName: delta}).
		Where("id = ?", subId).Exec(ctx)
	if err != nil {
		return fmt.Errorf("db stage score update failed, %w", err)
	}
	return nil
}

func (self *BunDatabase) GetScores(
	ctx context.Context,
	problemId int64,
	isAdmin bool,
) ([]models.UserScore, error) {
	var submissions []models.Submission
	var ret []models.UserScore
	query := self.db.NewSelect().DistinctOn("csid").Model(&submissions).
		Where("problemid = ?", problemId).
		OrderExpr("csid, id DESC")
	if isAdmin {
		query = query.Column("csid", "score_score", "score_stages")
	} else {
		query = query.Column("csid", "score_score")
	}
	err := query.Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("score query failed, %w", err)
	}

	for _, submission := range submissions {
		userScore := models.UserScore{
			CsId:  submission.CsId,
			Score: submission.Score,
		}
		if userScore.Score.Stages == nil {
			userScore.Score.Stages = make(map[string]int64)
		}
		ret = append(ret, userScore)
	}

	return ret, nil
}

func (self *BunDatabase) InsertUserResources(
	ctx context.Context,
	userResources []models.UserResource,
) error {
	for _, user := range userResources {
		_, err := self.db.NewInsert().Model(&user).
			On("CONFLICT (csid) DO UPDATE").
			Set("id = EXCLUDED.id").
			Set("roomid = EXCLUDED.roomid").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert user resource '%+v' failed, %w", user, err)
		}
	}
	return nil
}

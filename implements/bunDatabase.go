package implements

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type BunDatabase struct {
	db *bun.DB
}

func NewBunDatabase(db *bun.DB) *BunDatabase {
	return &BunDatabase{db: db}
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
		Relation("Jobs").Order("id DESC").Limit(1).Scan(ctx)
	return ret, err
}

func (self *BunDatabase) GetSubmission(
	ctx context.Context,
	subId int64,
) (models.Submission, error) {
	ret := models.Submission{Id: subId}
	err := self.db.NewSelect().Model(&ret).
		WherePK().Scan(ctx)
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
	delta int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Submission)(nil)).
		Set("score_score = score_score + ?", delta).
		Where("id = ?", subId).Exec(ctx)
	return err
}

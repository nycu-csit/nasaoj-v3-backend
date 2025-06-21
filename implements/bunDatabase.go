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

					Status: models.StatusBlocked,
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

func (self *BunDatabase) GetScoresWithStages(
	ctx context.Context,
	problemId int64,
) ([]models.UserScoreWithStages, error) {
	ret := []models.UserScoreWithStages{}
	var submissions []models.Submission
	err := self.db.NewSelect().DistinctOn("csid").Model(&submissions).
		Column("csid", "score_score", "score_stages").
		Where("problemid = ?", problemId).
		OrderExpr("csid, id DESC").
		Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("score query failed, %w", err)
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
	problemId int64,
) ([]models.UserScore, error) {
	ret := []models.UserScore{}
	var submissions []models.Submission
	err := self.db.NewSelect().DistinctOn("csid").Model(&submissions).
		Column("csid", "score_score").
		Where("problemid = ?", problemId).
		OrderExpr("csid, id DESC").
		Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("score query failed, %w", err)
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

func (self *BunDatabase) ExistUserResource(
	ctx context.Context,
	csId string,
) (bool, error) {
	return self.db.NewSelect().
		Model((*models.UserResource)(nil)).
		Where("csid = ?", csId).
		Exists(ctx)
}

func (self *BunDatabase) InsertNewUserFromResources(
	ctx context.Context,
	csId string,
) error {
	var resource models.Resource
	err := self.db.NewDelete().Model((*models.Resource)(nil)).
		Where("id IN (?)",
			self.db.NewSelect().Model((*models.Resource)(nil)).
				Column("id").Limit(1)).
		Returning("*").Scan(ctx, &resource)
	if err != nil {
		return fmt.Errorf("no user resource left, %w", err)
	}

	// Transform into UserResource
	user := resource.User(csId)
	_, err = self.db.NewInsert().Model(&user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert user failed, %w", err)
	}
	return nil
}

func (self *BunDatabase) GetSubmissionsAndCount(
	ctx context.Context,
	csId string,
	isCsId bool,
	pageNum int,
	isPageNum bool,
	pageSize int,
	problemId int64,
	isProblemId bool,
) ([]models.Submission, int, error) {
	query := self.db.NewSelect().Order("id DESC").Limit(pageSize)
	if isCsId {
		query = query.Where("csid = ?", csId)
	}
	if isPageNum {
		query = query.Offset(pageNum * pageSize)
	}
	if isProblemId {
		query = query.Where("problemid = ?", problemId)
	}

	var ret []models.Submission
	count, err := query.Model(&ret).ScanAndCount(ctx)
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

package implements

import (
	"context"
	"database/sql"
	"encoding/json"
	"text/template"
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
	err := self.db.NewSelect().
		Model(&ret).
		Where("cs_id = ?", csId).
		Where("ref_id = ?", refId).
		Where("type = ?", subType).
		Order("id DESC").
		Limit(1).
		Scan(ctx)
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

func (self *BunDatabase) InsertProblemSubmission(
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

				_, err = tx.NewInsert().Model(&job).Exec(ctx)
				if err != nil {
					return err
				}
			}
		}

		_, err = tx.NewUpdate().Model(submission).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (self *BunDatabase) InsertToolSubmission(
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
		Column("cs_id", "score_score", "score_stages").
		Where("ref_id = ?", refId).
		OrderExpr("cs_id, id DESC").
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
	err := self.db.NewSelect().DistinctOn("cs_id").Model(&submissions).
		Column("cs_id", "score_score").
		Where("ref_id = ?", refId).
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
	err := self.db.NewInsert().
		Model(&user).
		On("CONFLICT (args) DO UPDATE").
		Set("args = EXCLUDED.args").
		Scan(ctx)
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
	if isRefId {
		query = query.Where("ref_id = ?", refId)
	}
	if isRefType {
		query = query.Where("type = ?", refType)
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

func (self *BunDatabase) GetTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	var toolsMarshaled []models.ToolMarshaled
	err := self.db.NewSelect().
		Model(&toolsMarshaled).
		OrderExpr("tool_id, sha1").
		DistinctOn("tool_id").
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return map[string]models.Tool{}, err
	}

	ret := make(map[string]models.Tool)
	for _, tool := range toolsMarshaled {
		ret[tool.SHA1], err = tool.Unmarshal()
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal tool",
					zap.String("tool.sha1", tool.SHA1),
					zap.Error(err))
		}
	}
	return ret, nil
}

func (self *BunDatabase) GetProblem(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	var problemMarshaled models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemMarshaled).
		Where("sha1 = ?", sha1).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}

	return problemMarshaled.Unmarshal()
}

func (self *BunDatabase) GetTool(
	ctx context.Context,
	sha1 string,
) (models.Tool, error) {
	var toolMarshaled models.ToolMarshaled
	err := self.db.NewSelect().
		Model(&toolMarshaled).
		Where("sha1 = ?", sha1).
		Scan(ctx)
	if err != nil {
		return models.Tool{}, err
	}

	return toolMarshaled.Unmarshal()
}

func (self *BunDatabase) GetProblemById(
	ctx context.Context,
	problemId int64,
) (models.Problem, error) {
	var problemMarshaled models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemMarshaled).
		Where("problem_id = ?", problemId).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}

	return problemMarshaled.Unmarshal()
}

func (self *BunDatabase) GetToolById(
	ctx context.Context,
	toolId int64,
) (models.Tool, error) {
	var toolMarshaled models.ToolMarshaled
	err := self.db.NewSelect().
		Model(&toolMarshaled).
		Where("tool_id = ?", toolId).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return models.Tool{}, err
	}
	return toolMarshaled.Unmarshal()
}

func (self *BunDatabase) GetAllProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	var problemsMarshaled []models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemsMarshaled).
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

func (self *BunDatabase) GetAllTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	var toolsMarshaled []models.ToolMarshaled
	err := self.db.NewSelect().
		Model(&toolsMarshaled).
		Scan(ctx)
	if err != nil {
		return map[string]models.Tool{}, err
	}

	ret := make(map[string]models.Tool)
	for _, tool := range toolsMarshaled {
		ret[tool.SHA1], err = tool.Unmarshal()
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal tool",
					zap.String("tool.sha1", tool.SHA1),
					zap.Error(err))
			return map[string]models.Tool{}, err
		}
	}
	return ret, nil
}

func (self *BunDatabase) InsertProblem(
	ctx context.Context,
	problem models.Problem,
) error {
	problemMarshaled, err := problem.Marshal()
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal problem",
				zap.String("problem.name", problem.Name),
				zap.Error(err))
		return err
	}

	_, err = self.db.NewInsert().
		Model(&problemMarshaled).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	return err
}

func (self *BunDatabase) InsertContentFiles(
	ctx context.Context,
	contentFiles []models.ContentFile,
) error {
	if contentFiles == nil {
		return nil
	}
	_, err := self.db.NewInsert().
		Model(&contentFiles).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	return err
}

func (self *BunDatabase) GetContentFiles(
	ctx context.Context,
	sha1s []string,
) ([]models.ContentFile, error) {
	var res []models.ContentFile
	err := self.db.NewSelect().
		Model(&res).
		Where("sha1 IN (?)", bun.In(sha1s)).
		Scan(ctx)
	if err != nil {
		return []models.ContentFile{}, err
	}

	var ret []models.ContentFile
	for _, contentFile := range res {
		if contentFile.IsTemplate {
			contentFile.Template, err = template.New(contentFile.Name).
				Parse(contentFile.Payload)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to template file",
						zap.String("file.name", contentFile.Name),
						zap.Error(err))
				return []models.ContentFile{}, err
			}
		}
		ret = append(ret, contentFile)
	}
	return ret, nil
}

func (self *BunDatabase) GetAllContentFiles(
	ctx context.Context,
) ([]models.ContentFile, error) {
	var ret []models.ContentFile
	err := self.db.NewSelect().
		Model(&ret).
		Scan(ctx)
	if err != nil {
		return []models.ContentFile{}, err
	}

	for i, contentFile := range ret {
		if contentFile.IsTemplate {
			ret[i].Template, err = template.New(contentFile.Name).
				Parse(contentFile.Payload)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to template file",
						zap.String("file.name", contentFile.Name),
						zap.Error(err))
				return []models.ContentFile{}, err
			}
		}
	}
	return ret, nil
}

func (self *BunDatabase) InsertTool(
	ctx context.Context,
	tool models.Tool,
) error {
	toolMarshaled, err := tool.Marshal()
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal tool",
				zap.String("tool.name", tool.Name),
				zap.Error(err))
		return err
	}

	_, err = self.db.NewInsert().
		Model(&toolMarshaled).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	return err
}

func (self *BunDatabase) UpdateProblem(
	ctx context.Context,
	sha1 string,
	enable bool,
) error {
	if enable {
		return self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				problem := models.ProblemMarshaled{SHA1: sha1}
				err := tx.NewSelect().
					Column("sha1", "problem_id", "is_enabled").
					Model(&problem).
					WherePK().
					Scan(ctx)
				if err != nil {
					return err
				}
				_, err = tx.NewUpdate().
					Model((*models.ProblemMarshaled)(nil)).
					Set("is_enabled = CASE WHEN sha1 = ? THEN TRUE ELSE FALSE END", sha1).
					Where("problem_id = ?", problem.ProblemId).
					Exec(ctx)
				if err != nil {
					return err
				}
				return nil
			})
	} else {
		_, err := self.db.NewUpdate().
			Model((*models.ProblemMarshaled)(nil)).
			Where("sha1 = ?", sha1).
			Set("is_enabled = ?", false).
			Exec(ctx)
		return err
	}
}

func (self *BunDatabase) UpdateTool(
	ctx context.Context,
	sha1 string,
	enable bool,
) error {
	if enable {
		return self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				var tool models.ToolMarshaled
				err := tx.NewSelect().
					Model(&tool).
					Where("sha1 = ?", sha1).
					Scan(ctx)
				if err != nil {
					return err
				}
				_, err = tx.NewUpdate().
					Model((*models.ToolMarshaled)(nil)).
					Set("is_enabled = CASE WHEN sha1 = ? THEN TRUE ELSE FALSE END", sha1).
					Where("tool_id = ?", tool.ToolId).
					Exec(ctx)
				if err != nil {
					return err
				}
				return nil
			})
	} else {
		_, err := self.db.NewUpdate().
			Model((*models.ToolMarshaled)(nil)).
			Where("sha1 = ?", sha1).
			Set("is_enabled = ?", false).
			Exec(ctx)
		return err
	}
}

func (self *BunDatabase) HydrateContentFiles(
	ctx context.Context,
	contentFiles *[]models.ContentFile,
) error {
	var sha1s []string
	for _, file := range *contentFiles {
		sha1s = append(sha1s, file.SHA1)
	}

	res, err := self.GetContentFiles(ctx, sha1s)
	if err != nil {
		return err
	}
	*contentFiles = res
	return nil
}

func (self *BunDatabase) HydrateTool(
	ctx context.Context,
	tool *models.Tool,
) error {
	if err := self.HydrateContentFiles(
		ctx, &tool.PreScript.Files); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate tool prescript",
				zap.Error(err))
		return err
	}
	if err := self.HydrateContentFiles(
		ctx, &tool.PostScript.Files); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate tool postscript",
				zap.Error(err))
	}
	if err := self.HydrateContentFiles(
		ctx, &tool.Testcase.Content.Files); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate tool testcase content",
				zap.Error(err))
	}
	return nil
}

func (self *BunDatabase) HydrateProblem(
	ctx context.Context,
	problem *models.Problem,
) error {
	if err := self.HydrateContentFiles(
		ctx, &problem.PreScript.Files); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate problem prescript",
				zap.Error(err))
		return err
	}
	if err := self.HydrateContentFiles(
		ctx, &problem.PostScript.Files); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate problem postscript",
				zap.Error(err))
		return err
	}

	for stageName, stage := range problem.Stages {
		if err := self.HydrateContentFiles(
			ctx, &stage.PreScript.Files); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to hydrate stage prescript",
					zap.Error(err))
			return err
		}
		if err := self.HydrateContentFiles(
			ctx, &stage.PostScript.Files); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to hydrate stage postscript",
					zap.Error(err))
			return err
		}

		for i := range stage.Testcases {
			if err := self.HydrateContentFiles(ctx,
				&stage.Testcases[i].Content.Files); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to hydrate content files",
						zap.Error(err))
				return err
			}
		}
		problem.Stages[stageName] = stage
	}
	return nil
}

func (self *BunDatabase) EnableAllProblem(
	ctx context.Context,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.ProblemMarshaled)(nil)).
		Set("is_enabled = ?", true).
		Where("is_enabled = ?", false).
		Exec(ctx)
	return err
}

func (self *BunDatabase) DeleteProblemAndSubmissions(
	ctx context.Context,
	sha1 string,
) error {
	return self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewDelete().
				Model((*models.ProblemMarshaled)(nil)).
				Where("sha1 = ?", sha1).
				Exec(ctx)
			if err != nil {
				return err
			}
			_, err = tx.NewDelete().
				Model((*models.Submission)(nil)).
				Where("ref_sha1 = ?", sha1).
				Exec(ctx)
			if err != nil {
				return err
			}
			return nil
		})
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

func (self *BunDatabase) DeleteToolAndSubmissions(
	ctx context.Context,
	sha1 string,
) error {
	return self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewDelete().
				Model((*models.ToolMarshaled)(nil)).
				Where("sha1 = ?", sha1).
				Exec(ctx)
			if err != nil {
				return err
			}
			_, err = tx.NewDelete().
				Model((*models.Submission)(nil)).
				Where("ref_sha1 = ?", sha1).
				Exec(ctx)
			if err != nil {
				return err
			}
			return nil
		})
}

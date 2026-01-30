package implements

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"text/template"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

type Postgres struct {
	db    *bun.DB
	cache *cache.Cache
}

func NewPostgres(
	db *bun.DB,
	opt *cache.Options,
) *Postgres {
	var cash *cache.Cache = nil
	if opt != nil {
		cash = cache.New(opt)
	}
	return &Postgres{
		db:    db,
		cache: cash,
	}
}

func cacheContentFileSHA1Key(
	sha1 string,
) string {
	return "content-file-" + sha1
}

func (self *Postgres) getContentFile(
	ctx context.Context,
	sha1 string,
) (models.ContentFile, error) {
	var ret models.ContentFile
	err := self.db.NewSelect().
		Model(&ret).
		Where("sha1 = ?", sha1).
		Scan(ctx)
	if err != nil {
		return models.ContentFile{}, nil
	}

	if ret.IsTemplate {
		ret.Template, err = template.New(ret.Name).
			Parse(ret.Payload)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to template file",
					zap.String("file.name", ret.Name),
					zap.Error(err))
			return models.ContentFile{}, err
		}
	}

	return ret, nil
}

func (self *Postgres) GetContentFile(
	ctx context.Context,
	sha1 string,
) (models.ContentFile, error) {
	if self.cache != nil {
		var ret models.ContentFile
		err := self.cache.Once(&cache.Item{
			Key:   cacheContentFileSHA1Key(sha1),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getContentFile(ctx, sha1)
			},
		})
		if ret.IsTemplate {
			ret.Template, err = template.New(ret.Name).
				Parse(ret.Payload)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to template file",
						zap.String("file.name", ret.Name),
						zap.Error(err))
				return models.ContentFile{}, err
			}
		}
		return ret, err
	} else {
		return self.getContentFile(ctx, sha1)
	}
}

func (self *Postgres) GetContentFiles(
	ctx context.Context,
	sha1s []string,
) ([]models.ContentFile, error) {
	var ret []models.ContentFile
	for _, sha1 := range sha1s {
		contentFile, err := self.GetContentFile(ctx, sha1)
		if err != nil {
			return []models.ContentFile{}, err
		}
		ret = append(ret, contentFile)
	}
	return ret, nil
}

// No cache involved
func (self *Postgres) GetAllContentFiles(
	ctx context.Context,
) ([]models.ContentFile, error) {
	var ret []models.ContentFile
	err := self.db.NewSelect().
		Model(&ret).
		Scan(ctx)
	if err != nil {
		return []models.ContentFile{}, err
	}

	if err := self.HydrateContentFiles(ctx, &ret); err != nil {
		return []models.ContentFile{}, err
	}

	return ret, nil
}

func (self *Postgres) InsertContentFiles(
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

func (self *Postgres) HydrateContentFiles(
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

func (self *Postgres) HydrateProblem(
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

func (self *Postgres) HydrateTool(
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

func cacheProblemSHA1Key(
	sha1 string,
) string {
	return "problem-" + sha1
}

func cacheProblemIdKey(
	problemId int64,
) string {
	return "problem-id-" + strconv.FormatInt(problemId, 10)
}

func cacheProblemEnabledKey() string {
	return "problem-enabled"
}

func (self *Postgres) getProblem(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	ret := models.ProblemMarshaled{SHA1: sha1}
	err := self.db.NewSelect().
		Model(&ret).
		WherePK().
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}
	return ret.Unmarshal()
}

func (self *Postgres) GetProblem(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	if self.cache != nil {
		var ret models.Problem
		err := self.cache.Once(&cache.Item{
			Key:   cacheProblemSHA1Key(sha1),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getProblem(ctx, sha1)
			},
		})
		return ret, err
	} else {
		return self.getProblem(ctx, sha1)
	}
}

func (self *Postgres) getProblemById(
	ctx context.Context,
	problemId int64,
) (models.Problem, error) {
	var ret models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&ret).
		Where("problem_id = ?", problemId).
		Where("is_enabled = ?", true).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}
	return ret.Unmarshal()
}

func (self *Postgres) GetProblemById(
	ctx context.Context,
	problemId int64,
) (models.Problem, error) {
	if self.cache != nil {
		var problem models.Problem
		err := self.cache.Once(&cache.Item{
			Key:   cacheProblemIdKey(problemId),
			Value: &problem,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getProblemById(ctx, problemId)
			},
		})
		return problem, err
	} else {
		return self.getProblemById(ctx, problemId)
	}
}

func (self *Postgres) InsertProblem(
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

func (self *Postgres) getProblemsEnabled(
	ctx context.Context,
) ([]string, error) {
	var res []models.ProblemMarshaled
	err := self.db.NewSelect().
		Column("sha1", "is_enabled").
		Model(&res).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return []string{}, err
	}

	var ret []string
	for _, problem := range res {
		ret = append(ret, problem.SHA1)
	}
	return ret, nil
}

func (self *Postgres) GetProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	var problemsEnabled []string
	if self.cache != nil {
		err := self.cache.Once(&cache.Item{
			Key:   cacheProblemEnabledKey(),
			Value: &problemsEnabled,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getProblemsEnabled(ctx)
			},
		})
		if err != nil {
			return map[string]models.Problem{}, err
		}
	} else {
		var err error
		problemsEnabled, err = self.getProblemsEnabled(ctx)
		if err != nil {
			return map[string]models.Problem{}, err
		}
	}

	ret := make(map[string]models.Problem)
	for _, sha1 := range problemsEnabled {
		problem, err := self.GetProblem(ctx, sha1)
		if err != nil {
			return map[string]models.Problem{}, err
		}
		ret[sha1] = problem
	}
	return ret, nil
}

// No caching involved
func (self *Postgres) GetAllProblems(
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

func (self *Postgres) UpdateProblemState(
	ctx context.Context,
	sha1 string,
	isEnabled bool,
) error {
	if isEnabled {
		err := self.db.RunInTx(ctx, &sql.TxOptions{},
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
				return err
			})
		if err != nil {
			return err
		}
	} else {
		_, err := self.db.NewUpdate().
			Model((*models.ProblemMarshaled)(nil)).
			Where("sha1 = ?", sha1).
			Set("is_enabled = ?", false).
			Exec(ctx)
		if err != nil {
			return err
		}
	}
	if self.cache != nil {
		return self.cache.Delete(ctx, cacheProblemEnabledKey())
	} else {
		return nil
	}
}

func (self *Postgres) GetProblemScores(
	ctx context.Context,
	sha1s []string,
) (map[string]int64, error) {
	ret := make(map[string]int64)
	for _, sha1 := range sha1s {
		problem, err := self.GetProblem(ctx, sha1)
		if err != nil {
			return map[string]int64{}, err
		}

		ret[sha1] = problem.TotalScore
	}
	return ret, nil
}

func (self *Postgres) DeleteProblemAndSubmissions(
	ctx context.Context,
	sha1 string,
) error {
	problemMarshaled := models.ProblemMarshaled{SHA1: sha1}
	err := self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewDelete().
				Model(&problemMarshaled).
				WherePK().
				Returning("problem_id").
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
	if err != nil {
		return err
	}

	problem, err := problemMarshaled.Unmarshal()
	if err != nil {
		return err
	}

	if self.cache != nil {
		if err := self.cache.Delete(ctx,
			cacheProblemIdKey(problem.ProblemId)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			cacheProblemSHA1Key(sha1)); err != nil {
			return err
		}
	}
	return nil
}

func cacheToolSHA1Key(
	sha1 string,
) string {
	return "tool-" + sha1
}

func cacheToolIdKey(
	toolId int64,
) string {
	return "tool-id-" + strconv.FormatInt(toolId, 10)
}

func cacheToolEnabledKey() string {
	return "tool-enabled"
}

func (self *Postgres) getTool(
	ctx context.Context,
	sha1 string,
) (models.Tool, error) {
	ret := models.ToolMarshaled{SHA1: sha1}
	err := self.db.NewSelect().
		Model(&ret).
		WherePK().
		Scan(ctx)
	if err != nil {
		return models.Tool{}, err
	}
	return ret.Unmarshal()
}

func (self *Postgres) GetTool(
	ctx context.Context,
	sha1 string,
) (models.Tool, error) {
	if self.cache != nil {
		var ret models.Tool
		err := self.cache.Once(&cache.Item{
			Key:   cacheToolSHA1Key(sha1),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getTool(ctx, sha1)
			},
		})
		return ret, err
	} else {
		return self.getTool(ctx, sha1)
	}
}

func (self *Postgres) getToolById(
	ctx context.Context,
	toolId int64,
) (models.Tool, error) {
	var ret models.ToolMarshaled
	err := self.db.NewSelect().
		Model(&ret).
		Where("tool_id = ?", toolId).
		Where("is_enabled = ?", true).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return models.Tool{}, err
	}
	return ret.Unmarshal()
}

func (self *Postgres) GetToolById(
	ctx context.Context,
	toolId int64,
) (models.Tool, error) {
	if self.cache != nil {
		var ret models.Tool
		err := self.cache.Once(&cache.Item{
			Key:   cacheToolIdKey(toolId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getToolById(ctx, toolId)
			},
		})
		return ret, err
	} else {
		return self.getToolById(ctx, toolId)
	}
}

func (self *Postgres) InsertTool(
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

func (self *Postgres) getToolsEnabled(
	ctx context.Context,
) ([]string, error) {
	var res []models.ToolMarshaled
	err := self.db.NewSelect().
		Column("sha1", "is_enabled").
		Model(&res).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return []string{}, err
	}

	var ret []string
	for _, tool := range res {
		ret = append(ret, tool.SHA1)
	}
	return ret, nil
}

func (self *Postgres) GetTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	var toolsEnabled []string
	if self.cache != nil {
		err := self.cache.Once(&cache.Item{
			Key:   cacheToolEnabledKey(),
			Value: &toolsEnabled,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getToolsEnabled(ctx)
			},
		})
		if err != nil {
			return map[string]models.Tool{}, err
		}
	} else {
		var err error
		toolsEnabled, err = self.getToolsEnabled(ctx)
		if err != nil {
			return map[string]models.Tool{}, err
		}
	}

	ret := make(map[string]models.Tool)
	for _, sha1 := range toolsEnabled {
		tool, err := self.GetTool(ctx, sha1)
		if err != nil {
			return map[string]models.Tool{}, err
		}
		ret[sha1] = tool
	}
	return ret, nil
}

// No caching involved
func (self *Postgres) GetAllTools(
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

func (self *Postgres) UpdateToolState(
	ctx context.Context,
	sha1 string,
	isEnabled bool,
) error {
	if isEnabled {
		err := self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				tool := models.ToolMarshaled{SHA1: sha1}
				err := tx.NewSelect().
					Column("sha1", "tool_id", "is_enabled").
					Model(&tool).
					WherePK().
					Scan(ctx)
				if err != nil {
					return err
				}
				_, err = tx.NewUpdate().
					Model((*models.ToolMarshaled)(nil)).
					Set("is_enabled = CASE WHEN sha1 = ? THEN TRUE ELSE FALSE END", sha1).
					Where("tool_id = ?", tool.ToolId).
					Exec(ctx)
				return err
			})
		if err != nil {
			return err
		}
	} else {
		_, err := self.db.NewUpdate().
			Model((*models.ToolMarshaled)(nil)).
			Where("sha1 = ?", sha1).
			Set("is_enabled = ?", false).
			Exec(ctx)
		if err != nil {
			return err
		}
	}
	if self.cache != nil {
		return self.cache.Delete(ctx, cacheToolEnabledKey())
	} else {
		return nil
	}
}

func (self *Postgres) DeleteToolAndSubmissions(
	ctx context.Context,
	sha1 string,
) error {
	toolMarshaled := models.ToolMarshaled{SHA1: sha1}
	err := self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewDelete().
				Model(&toolMarshaled).
				WherePK().
				Returning("tool_id").
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
	if err != nil {
		return err
	}

	tool, err := toolMarshaled.Unmarshal()
	if err != nil {
		return err
	}

	if self.cache != nil {
		if err := self.cache.Delete(ctx,
			cacheToolIdKey(tool.ToolId)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			cacheToolSHA1Key(sha1)); err != nil {
			return err
		}
	}
	return nil
}

func cacheScoresKey(
	problemId int64,
) string {
	return "scores-" + strconv.FormatInt(problemId, 10)
}

func cacheScoresWStagesKey(
	problemId int64,
) string {
	return "scores-w-stages-" + strconv.FormatInt(problemId, 10)
}

func cacheUserScoresKey(
	problemId int64,
	csId string,
) string {
	return "user-scores-" + strconv.FormatInt(problemId, 10) + "-" + csId
}

func cacheUsersKey(
	problemId int64,
) string {
	return "users-" + strconv.FormatInt(problemId, 10)
}

func cacheUserStageScoresKey(
	problemId int64,
	csId string,
) string {
	return "user-stage-scores-" + strconv.FormatInt(problemId, 10) + "-" + csId
}

func (self *Postgres) getUserScoreWithStages(
	ctx context.Context,
	refId int64,
	csId string,
) (models.UserScoreWithStages, error) {
	var sub models.Submission
	err := self.db.NewSelect().
		Model(&sub).
		Column("id", "cs_id", "problem_id", "problem_sha1").
		Where("problem_id = ?", refId).
		Where("cs_id = ?", csId).
		OrderExpr("id DESC").
		Limit(1).
		Relation("Jobs").
		Scan(ctx)
	if err != nil {
		return models.UserScoreWithStages{}, err
	}

	problem, err := self.GetProblem(ctx, sub.ProblemSHA1)
	if err != nil {
		return models.UserScoreWithStages{}, err
	}

	return models.UserScoreWithStages{
		CsId:      csId,
		Score:     utils.CalcScore(problem, sub),
		CreatedAt: sub.CreatedAt,
	}, nil
}

func (self *Postgres) GetUserScoreWithStages(
	ctx context.Context,
	refId int64,
	csId string,
) (models.UserScoreWithStages, error) {
	if self.cache != nil {
		var ret models.UserScoreWithStages
		err := self.cache.Once(&cache.Item{
			Key:   cacheUserScoresKey(refId, csId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getUserScoreWithStages(ctx, refId, csId)
			},
		})
		return ret, err
	} else {
		return self.getUserScoreWithStages(ctx, refId, csId)
	}
}

func (self *Postgres) getSubmittedUsers(
	ctx context.Context,
	refId int64,
) ([]string, error) {
	var subs []models.Submission
	err := self.db.NewSelect().
		Model(&subs).
		DistinctOn("cs_id").
		Column("cs_id").
		Where("problem_id = ?", refId).
		Scan(ctx)
	if err != nil {
		return []string{}, err
	}

	var ret []string
	for _, sub := range subs {
		ret = append(ret, sub.CsId)
	}
	return ret, nil
}

func (self *Postgres) GetSubmittedUsers(
	ctx context.Context,
	refId int64,
) ([]string, error) {
	if self.cache != nil {
		var ret []string
		err := self.cache.Once(&cache.Item{
			Key:   cacheUsersKey(refId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getSubmittedUsers(ctx, refId)
			},
		})
		return ret, err
	} else {
		return self.getSubmittedUsers(ctx, refId)
	}
}

func (self *Postgres) getScores(
	ctx context.Context,
	refId int64,
) ([]models.UserScore, error) {
	users, err := self.GetSubmittedUsers(ctx, refId)
	if err != nil {
		return []models.UserScore{}, err
	}

	var ret []models.UserScore
	for _, user := range users {
		score, err := self.GetUserScoreWithStages(ctx, refId, user)
		if err != nil {
			return []models.UserScore{}, err
		}
		ret = append(ret, models.UserScore{
			CsId:      score.CsId,
			Score:     score.Score.Score,
			CreatedAt: score.CreatedAt,
		})
	}
	return ret, nil
}

func (self *Postgres) GetScores(
	ctx context.Context,
	refId int64,
) ([]models.UserScore, error) {
	if self.cache != nil {
		var ret []models.UserScore
		err := self.cache.Once(&cache.Item{
			Key:   cacheScoresKey(refId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getScores(ctx, refId)
			},
		})
		return ret, err
	} else {
		return self.getScores(ctx, refId)
	}
}

func (self *Postgres) getScoresWithStages(
	ctx context.Context,
	refId int64,
) ([]models.UserScoreWithStages, error) {
	users, err := self.GetSubmittedUsers(ctx, refId)
	if err != nil {
		return []models.UserScoreWithStages{}, err
	}

	var ret []models.UserScoreWithStages
	for _, user := range users {
		score, err := self.GetUserScoreWithStages(ctx, refId, user)
		if err != nil {
			return []models.UserScoreWithStages{}, err
		}
		ret = append(ret, score)
	}
	return ret, nil
}

func (self *Postgres) GetScoresWithStages(
	ctx context.Context,
	refId int64,
) ([]models.UserScoreWithStages, error) {
	if self.cache != nil {
		var ret []models.UserScoreWithStages
		err := self.cache.Once(&cache.Item{
			Key:   cacheScoresWStagesKey(refId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getScoresWithStages(ctx, refId)
			},
		})
		return ret, err
	} else {
		return self.getScoresWithStages(ctx, refId)
	}
}

func (self *Postgres) GetScoresWithDeadline(
	ctx context.Context,
	refId int64,
) ([]models.UserScore, error) {
	var submissions []models.Submission
	var problem models.Problem
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var problemMarshaled models.ProblemMarshaled
		err := tx.NewSelect().
			Model(&problemMarshaled).
			Where("problem_id = ?", refId).
			Where("is_enabled = ?", true).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return err
		}

		problem, err = problemMarshaled.Unmarshal()
		if err != nil {
			return err
		}

		validSubQ := tx.NewSelect().
			Model((*models.Submission)(nil)).
			DistinctOn("cs_id").
			Column("cs_id", "created_at", "id").
			Where("problem_id = ?", refId).
			Order("cs_id ASC", "created_at DESC")

		if problem.Deadline != 0 {
			validSubQ = validSubQ.
				Where("created_at < ?", problem.Deadline)
		}

		err = self.db.NewSelect().
			With("valid_sub", validSubQ).
			Model((*models.Submission)(nil)).
			Where("id IN (?)", tx.NewSelect().
				Table("valid_sub").
				Column("id")).
			Order("cs_id ASC").
			Relation("Jobs").
			Scan(ctx, &submissions)

		return err
	})
	if err != nil {
		return []models.UserScore{}, err
	}

	var ret []models.UserScore
	for _, sub := range submissions {
		score := utils.CalcScore(problem, sub)

		ret = append(ret, models.UserScore{
			Score:     score.Score,
			CsId:      sub.CsId,
			CreatedAt: sub.CreatedAt,
		})
	}

	return ret, nil
}

func (self *Postgres) submissionQueryHook(
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

func (self *Postgres) GetSubmission(
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

func (self *Postgres) GetLastSubmission(
	ctx context.Context,
	csId string,
	refId int64,
	subType models.RefType,
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

func (self *Postgres) GetSubmissionsAndCount(
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

func (self *Postgres) DeleteSubmission(
	ctx context.Context,
	subId int64,
) error {
	var sub models.Submission
	_, err := self.db.NewDelete().
		Model((*models.Submission)(nil)).
		Where("id = ?", subId).
		Returning("problem_id", "cs_id").
		Exec(ctx, &sub)
	if err != nil {
		return err
	}

	pending := []string{
		cacheUserScoresKey(sub.ProblemId, sub.CsId),
		cacheUserStageScoresKey(sub.ProblemId, sub.CsId),
		cacheUsersKey(sub.ProblemId),
		cacheScoresKey(sub.ProblemId),
		cacheScoresWStagesKey(sub.ProblemId),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (self *Postgres) InsertSubmissionByProblem(
	ctx context.Context,
	submission *models.Submission,
	problem models.Problem,
) error {
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
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
	if err != nil {
		return err
	}

	pending := []string{
		cacheUsersKey(submission.Id),
		cacheScoresKey(submission.Id),
		cacheScoresWStagesKey(submission.Id),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (self *Postgres) InsertSubmissionByTool(
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

func (self *Postgres) DeleteUserAndSubmissions(
	ctx context.Context,
	csId string,
) error {
	var subs []models.Submission
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
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
			Returning("problem_id").
			Exec(ctx, &subs)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	var pending []string

	for _, sub := range subs {
		pending = append(pending, cacheUserScoresKey(sub.ProblemId, csId))
		pending = append(pending, cacheUsersKey(sub.ProblemId))
		pending = append(pending, cacheScoresKey(sub.ProblemId))
		pending = append(pending, cacheScoresWStagesKey(sub.ProblemId))
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (self *Postgres) SkipJobs(
	ctx context.Context,
	subId int64,
) error {
	var sub models.Submission
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().
			Model(&sub).
			Where("sub_id = ?", subId).
			Scan(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewUpdate().
			Model((*models.Job)(nil)).
			Set("status = ?", models.StatusSkip).
			Where("sub_id = ?", subId).
			WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
				return q.
					Where("status = ?", models.StatusBlocked).
					WhereOr("status = ?", models.StatusPending)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	pending := []string{
		cacheUsersKey(sub.ProblemId),
		cacheScoresKey(sub.ProblemId),
		cacheScoresWStagesKey(sub.ProblemId),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (self *Postgres) UpdateJob(
	ctx context.Context,
	job *models.Job,
) error {
	var sub models.Submission
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().
			Model(&sub).
			Where("id = ?", job.SubId).
			Scan(ctx)
		if err != nil {
			return err
		}
		_, err = tx.NewUpdate().
			Model(job).
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	pending := []string{
		cacheUsersKey(sub.ProblemId),
		cacheScoresKey(sub.ProblemId),
		cacheScoresWStagesKey(sub.ProblemId),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (self *Postgres) GetUserResource(
	ctx context.Context,
	csId string,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().Model(&ret).
		Where("cs_id = ?", csId).Scan(ctx)
	return ret, err
}

func (self *Postgres) InsertUserResources(
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

func (self *Postgres) ExistUserResource(
	ctx context.Context,
	csId string,
) (bool, error) {
	return self.db.NewSelect().
		Model((*models.UserResource)(nil)).
		Where("cs_id = ?", csId).
		Exists(ctx)
}

func (self *Postgres) UpsertUser(
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

func (self *Postgres) GetUsers(
	ctx context.Context,
) ([]models.UserResource, error) {
	var ret []models.UserResource
	err := self.db.NewSelect().Model(&ret).Scan(ctx)
	return ret, err
}

func (self *Postgres) GetSession(
	ctx context.Context,
	csId string,
) (models.Session, error) {
	ret := models.Session{CsId: csId}
	err := self.db.NewSelect().Model(&ret).WherePK().Scan(ctx)
	return ret, err
}

func (self *Postgres) UpsertSession(
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

func (self *Postgres) UpdateRefreshToken(
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

func (self *Postgres) InvalidateSession(
	ctx context.Context,
	csId string,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Session)(nil)).
		Where("cs_id = ?", csId).
		Set("is_valid = ?", false).
		Exec(ctx)
	return err
}

func (self *Postgres) DeleteUser(
	ctx context.Context,
	csId string,
) error {
	_, err := self.db.NewDelete().
		Model((*models.UserResource)(nil)).
		Where("cs_id = ?", csId).
		Exec(ctx)
	return err
}

package implements

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cache"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
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

	for i, file := range *contentFiles {
		res[i].Name = file.Name
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

func (self *Postgres) DehydrateContentFiles(
	contentFiles *[]models.ContentFile,
) {
	for i := range *contentFiles {
		(*contentFiles)[i].Payload = ""
		(*contentFiles)[i].IsTemplate = false
		(*contentFiles)[i].Template = nil
	}
}

func (self *Postgres) DehydrateProblem(
	problem *models.Problem,
) {
	self.DehydrateContentFiles(&problem.PreScript.Files)
	self.DehydrateContentFiles(&problem.PostScript.Files)

	for stageName, stage := range problem.Stages {
		self.DehydrateContentFiles(&stage.PreScript.Files)
		self.DehydrateContentFiles(&stage.PostScript.Files)

		for i := range stage.Testcases {
			self.DehydrateContentFiles(&stage.Testcases[i].Content.Files)
		}
		problem.Stages[stageName] = stage
	}
}

func (self *Postgres) DehydrateTool(
	tool *models.Tool,
) {
	self.DehydrateContentFiles(&tool.PreScript.Files)
	self.DehydrateContentFiles(&tool.PostScript.Files)
	self.DehydrateContentFiles(&tool.Testcase.Content.Files)
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
) (string, error) {
	problemMarshaled, err := problem.Marshal()
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal problem",
				zap.String("problem.name", problem.Name),
				zap.Error(err))
		return "", err
	}

	_, err = self.db.NewInsert().
		Model(&problemMarshaled).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return "", err
	}

	if self.cache != nil {
		if err := self.cache.Delete(
			ctx, cacheProblemSHA1Key(problemMarshaled.SHA1)); err != nil {
			return "", err
		}
		if err := self.cache.Delete(ctx, cacheProblemEnabledKey()); err != nil {
			return "", err
		}
	}
	return problemMarshaled.SHA1, nil
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
	sha1Prefix string,
	isEnabled bool,
) error {
	var sha1 string
	if isEnabled {
		err := self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				var problem models.ProblemMarshaled
				err := tx.NewSelect().
					Column("sha1", "problem_id", "is_enabled").
					Model(&problem).
					Where("sha1 LIKE ?", sha1Prefix+"%").
					Scan(ctx)
				if err != nil {
					return err
				}
				sha1 = problem.SHA1
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
		var problem models.ProblemMarshaled
		_, err := self.db.NewUpdate().
			Model(&problem).
			Where("sha1 = (?)", self.db.NewSelect().
				Column("sha1").
				Model((*models.ProblemMarshaled)(nil)).
				Where("sha1 LIKE ?", sha1Prefix+"%").
				Limit(1)).
			Set("is_enabled = ?", false).
			Returning("sha1").
			Exec(ctx)
		if err != nil {
			return err
		}
		sha1 = problem.SHA1
	}
	if self.cache != nil {
		if err := self.cache.Delete(
			ctx, cacheProblemSHA1Key(sha1)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx, cacheProblemEnabledKey()); err != nil {
			return err
		}
	}
	return nil
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
	sha1Prefix string,
) error {
	var isDeleted bool
	var problem models.ProblemMarshaled
	err := self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			res, err := tx.NewDelete().
				Model(&problem).
				Where("sha1 = (?)", tx.NewSelect().
					Model((*models.ProblemMarshaled)(nil)).
					Column("sha1").
					Where("sha1 LIKE ?", sha1Prefix+"%").
					Limit(1)).
				Returning("sha1", "problem_id").
				Exec(ctx)
			if err != nil {
				return err
			}
			cnt, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if cnt == 0 {
				isDeleted = false
				return nil
			}
			_, err = tx.NewDelete().
				Model((*models.Submission)(nil)).
				Where("problem_sha1 = ?", problem.SHA1).
				Exec(ctx)
			if err != nil {
				return err
			}
			isDeleted = true
			return nil
		})
	if err != nil {
		return err
	}

	if isDeleted && self.cache != nil {
		if err := self.cache.Delete(ctx,
			cacheProblemIdKey(problem.ProblemId)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			cacheProblemSHA1Key(problem.SHA1)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			cacheProblemEnabledKey()); err != nil {
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
) (string, error) {
	toolMarshaled, err := tool.Marshal()
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal tool",
				zap.String("tool.name", tool.Name),
				zap.Error(err))
		return "", err
	}
	_, err = self.db.NewInsert().
		Model(&toolMarshaled).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return "", err
	}

	if self.cache != nil {
		if err := self.cache.Delete(
			ctx, cacheToolSHA1Key(toolMarshaled.SHA1)); err != nil {
			return "", err
		}
		if err := self.cache.Delete(ctx, cacheToolEnabledKey()); err != nil {
			return "", err
		}
	}
	return toolMarshaled.SHA1, nil
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
	sha1Prefix string,
	isEnabled bool,
) error {
	var sha1 string
	if isEnabled {
		err := self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				var tool models.ToolMarshaled
				err := tx.NewSelect().
					Column("sha1", "tool_id", "is_enabled").
					Model(&tool).
					Where("sha1 LIKE ?", sha1Prefix+"%").
					Scan(ctx)
				if err != nil {
					return err
				}
				sha1 = tool.SHA1
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
		var tool models.ToolMarshaled
		_, err := self.db.NewUpdate().
			Model(&tool).
			Where("sha1 = (?)", self.db.NewSelect().
				Column("sha1").
				Model((*models.ToolMarshaled)(nil)).
				Where("sha1 LIKE ?", sha1Prefix+"%").
				Limit(1)).
			Set("is_enabled = ?", false).
			Returning("sha1").
			Exec(ctx)
		if err != nil {
			return err
		}
		sha1 = tool.SHA1
	}
	if self.cache != nil {
		if err := self.cache.Delete(
			ctx, cacheToolSHA1Key(sha1)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx, cacheToolEnabledKey()); err != nil {
			return err
		}
	}
	return nil
}

func (self *Postgres) DeleteToolAndSubmissions(
	ctx context.Context,
	sha1Prefix string,
) error {
	var isDeleted bool
	var tool models.ToolMarshaled
	err := self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			res, err := tx.NewDelete().
				Model(&tool).
				Where("sha1 = (?)", tx.NewSelect().
					Model((*models.ToolMarshaled)(nil)).
					Column("sha1").
					Where("sha1 LIKE ?", sha1Prefix+"%").
					Limit(1)).
				Returning("sha1", "tool_id").
				Exec(ctx)
			if err != nil {
				return err
			}
			cnt, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if cnt == 0 {
				isDeleted = false
				return nil
			}
			_, err = tx.NewDelete().
				Model((*models.Submission)(nil)).
				Where("tool_sha1 = ?", tool.SHA1).
				Exec(ctx)
			if err != nil {
				return err
			}
			isDeleted = true
			return nil
		})
	if err != nil {
		return err
	}

	if isDeleted && self.cache != nil {
		if err := self.cache.Delete(ctx,
			cacheToolIdKey(tool.ToolId)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			cacheToolSHA1Key(tool.SHA1)); err != nil {
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
	id int64,
) string {
	return "user-scores-" + strconv.FormatInt(problemId, 10) + "-" + strconv.FormatInt(id, 10)
}

func cacheSubmissionScoreKey(
	subId int64,
) string {
	return "sub-scores-" + strconv.FormatInt(subId, 10)
}

func cacheUsersKey(
	problemId int64,
) string {
	return "users-" + strconv.FormatInt(problemId, 10)
}

func cacheUserStageScoresKey(
	problemId int64,
	id int64,
) string {
	return "user-stage-scores-" + strconv.FormatInt(problemId, 10) + "-" + strconv.FormatInt(id, 10)
}

func (self *Postgres) getSubmissionScore(
	ctx context.Context,
	subId int64,
) (models.UserScore, error) {
	sub := models.Submission{Id: subId}
	err := self.db.NewSelect().
		Model(&sub).
		WherePK().
		Relation("Jobs").
		Scan(ctx)
	if err != nil {
		return models.UserScore{}, err
	}

	problem, err := self.GetProblem(ctx, sub.ProblemSHA1)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get problem",
				zap.Error(err))
		return models.UserScore{}, err
	}

	user, err := self.GetUserResource(ctx, sub.OwnerId)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get user resource",
				zap.Error(err))
		return models.UserScore{}, err
	}

	var score int64
	var isFinal bool

	if problem.ScoreType == models.ScoreTypeSelfManaged {
		// For self-managed scoring, use the score stored in submission
		score = sub.Score
		// isFinal is true when all jobs are done (not pending/running/blocked)
		isFinal = true
		for _, job := range sub.Jobs {
			switch job.Status {
			case models.StatusPending, models.StatusRunning, models.StatusBlocked:
				isFinal = false
			}
		}
	} else {
		// Default: calculate from job results
		calcScore, final := utils.CalcScore(problem, sub)
		score = calcScore.Score
		isFinal = final
	}

	return models.UserScore{
		Score:    score,
		Id:       sub.OwnerId,
		Username: user.Username,

		CreatedAt: sub.CreatedAt,
		IsFinal:   isFinal,
	}, nil
}

func (self *Postgres) GetSubmissionScore(
	ctx context.Context,
	subId int64,
) (models.UserScore, error) {
	if self.cache != nil {
		var ret models.UserScore
		err := self.cache.Once(&cache.Item{
			Key:   cacheSubmissionScoreKey(subId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getSubmissionScore(ctx, subId)
			},
			TTL: 2 * time.Second,
		})
		if ret.IsFinal {
			err = self.cache.Expire(ctx, cacheSubmissionScoreKey(subId), time.Hour)
			if err != nil {
				return models.UserScore{}, err
			}
		}
		return ret, err
	} else {
		return self.getSubmissionScore(ctx, subId)
	}
}

func (self *Postgres) getUserScoreWithStages(
	ctx context.Context,
	refId int64,
	id int64,
) (models.UserScoreWithStages, error) {
	var sub models.Submission
	err := self.db.NewSelect().
		Model(&sub).
		Column("id", "owner_id", "problem_id", "problem_sha1", "created_at", "score").
		Where("problem_id = ?", refId).
		Where("owner_id = ?", id).
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

	user, err := self.GetUserResource(ctx, id)
	if err != nil {
		return models.UserScoreWithStages{}, err
	}

	var scoreResult models.Score
	var isFinal bool

	if problem.ScoreType == models.ScoreTypeSelfManaged {
		// For self-managed scoring, return the stored score with empty stages
		scoreResult = models.Score{
			Score:  sub.Score,
			Stages: make(map[string]int64),
		}
		isFinal = true
		for _, job := range sub.Jobs {
			switch job.Status {
			case models.StatusPending, models.StatusRunning, models.StatusBlocked:
				isFinal = false
			}
		}
	} else {
		scoreResult, isFinal = utils.CalcScore(problem, sub)
	}

	return models.UserScoreWithStages{
		Id:       id,
		Username: user.Username,
		Score:    scoreResult,

		CreatedAt: sub.CreatedAt,
		IsFinal:   isFinal,
	}, nil
}

func (self *Postgres) GetUserScoreWithStages(
	ctx context.Context,
	refId int64,
	id int64,
) (models.UserScoreWithStages, error) {
	if self.cache != nil {
		var ret models.UserScoreWithStages
		err := self.cache.Once(&cache.Item{
			Key:   cacheUserScoresKey(refId, id),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getUserScoreWithStages(ctx, refId, id)
			},
		})
		if ret.IsFinal {
			err = self.cache.Expire(ctx, cacheUserScoresKey(refId, id), time.Hour)
			if err != nil {
				return models.UserScoreWithStages{}, err
			}
		}
		return ret, err
	} else {
		return self.getUserScoreWithStages(ctx, refId, id)
	}
}

func (self *Postgres) getSubmittedUsers(
	ctx context.Context,
	refId int64,
) ([]int64, error) {
	var subs []models.Submission
	err := self.db.NewSelect().
		Model(&subs).
		DistinctOn("owner_id").
		Column("owner_id").
		Where("problem_id = ?", refId).
		Scan(ctx)
	if err != nil {
		return []int64{}, err
	}

	var ret []int64
	for _, sub := range subs {
		ret = append(ret, sub.OwnerId)
	}
	return ret, nil
}

func (self *Postgres) GetSubmittedUsers(
	ctx context.Context,
	refId int64,
) ([]int64, error) {
	if self.cache != nil {
		var ret []int64
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
) ([]models.UserScore, bool, error) {
	users, err := self.GetSubmittedUsers(ctx, refId)
	if err != nil {
		return []models.UserScore{}, false, err
	}

	isFinal := true
	var ret []models.UserScore
	for _, user := range users {
		score, err := self.GetUserScoreWithStages(ctx, refId, user)
		if err != nil {
			return []models.UserScore{}, false, err
		}
		ret = append(ret, models.UserScore{
			Id:       score.Id,
			Username: score.Username,
			Score:    score.Score.Score,

			CreatedAt: score.CreatedAt,
		})
		if !score.IsFinal {
			isFinal = false
		}
	}
	return ret, isFinal, nil
}

func (self *Postgres) GetScores(
	ctx context.Context,
	refId int64,
) ([]models.UserScore, error) {
	if self.cache != nil {
		var ret []models.UserScore
		var isFinal bool
		err := self.cache.Once(&cache.Item{
			Key:   cacheScoresKey(refId),
			Value: &ret,
			Do: func(*cache.Item) (ret interface{}, err error) {
				ret, isFinal, err = self.getScores(ctx, refId)
				return
			},
		})
		if isFinal {
			err := self.cache.Expire(ctx, cacheScoresKey(refId), time.Hour)
			if err != nil {
				return []models.UserScore{}, err
			}
		}
		return ret, err
	} else {
		ret, _, err := self.getScores(ctx, refId)
		return ret, err
	}
}

func (self *Postgres) getScoresWithStages(
	ctx context.Context,
	refId int64,
) ([]models.UserScoreWithStages, bool, error) {
	users, err := self.GetSubmittedUsers(ctx, refId)
	if err != nil {
		return []models.UserScoreWithStages{}, false, err
	}

	isFinal := true
	var ret []models.UserScoreWithStages
	for _, user := range users {
		score, err := self.GetUserScoreWithStages(ctx, refId, user)
		if err != nil {
			return []models.UserScoreWithStages{}, false, err
		}
		ret = append(ret, score)

		if !score.IsFinal {
			isFinal = false
		}
	}
	return ret, isFinal, nil
}

func (self *Postgres) GetScoresWithStages(
	ctx context.Context,
	refId int64,
) ([]models.UserScoreWithStages, error) {
	if self.cache != nil {
		var ret []models.UserScoreWithStages
		var isFinal bool
		err := self.cache.Once(&cache.Item{
			Key:   cacheScoresWStagesKey(refId),
			Value: &ret,
			Do: func(*cache.Item) (ret interface{}, err error) {
				ret, isFinal, err = self.getScoresWithStages(ctx, refId)
				return
			},
		})
		if isFinal {
			err := self.cache.Expire(ctx, cacheScoresWStagesKey(refId), time.Hour)
			if err != nil {
				return []models.UserScoreWithStages{}, err
			}
		}
		return ret, err
	} else {
		ret, _, err := self.getScoresWithStages(ctx, refId)
		return ret, err
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
			DistinctOn("owner_id").
			Column("owner_id", "created_at", "id").
			Where("problem_id = ?", refId).
			Order("owner_id ASC", "created_at DESC")

		if problem.Deadline != 0 {
			validSubQ = validSubQ.
				Where("created_at < ?", problem.Deadline)
		}

		err = self.db.NewSelect().
			With("valid_sub", validSubQ).
			Model(&submissions).
			Where("id IN (?)", tx.NewSelect().
				Table("valid_sub").
				Column("id")).
			Order("owner_id ASC").
			Relation("Jobs").
			Scan(ctx)

		return err
	})
	if err != nil {
		return []models.UserScore{}, err
	}

	var ret []models.UserScore
	for _, sub := range submissions {
		var score int64
		if problem.ScoreType == models.ScoreTypeSelfManaged {
			score = sub.Score
		} else {
			calcScore, _ := utils.CalcScore(problem, sub)
			score = calcScore.Score
		}

		user, err := self.GetUserResource(ctx, sub.OwnerId)
		if err != nil {
			return []models.UserScore{}, err
		}

		ret = append(ret, models.UserScore{
			Score:     score,
			Id:        sub.OwnerId,
			Username:  user.Username,
			CreatedAt: sub.CreatedAt,
		})
	}

	return ret, nil
}

func (self *Postgres) GetSubmission(
	ctx context.Context,
	subId int64,
) (models.Submission, error) {
	ret := models.Submission{Id: subId}

	err := self.db.NewSelect().
		Model(&ret).
		WherePK().
		Relation("Jobs").
		Scan(ctx)
	return ret, err
}

func (self *Postgres) GetLastSubmission(
	ctx context.Context,
	id int64,
	refId int64,
	subType models.RefType,
) (models.Submission, error) {
	var ret models.Submission
	query := self.db.NewSelect().
		Model(&ret).
		Where("owner_id = ?", id).
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
	id int64,
	isOwnerId bool,
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
	if isOwnerId {
		query = query.Where("owner_id = ?", id)
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
		Returning("problem_id", "owner_id").
		Exec(ctx, &sub)
	if err != nil {
		return err
	}

	pending := []string{
		cacheUserScoresKey(sub.ProblemId, sub.OwnerId),
		cacheUserStageScoresKey(sub.ProblemId, sub.OwnerId),
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
	id int64,
) error {
	var subs []models.Submission
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().
			Model((*models.UserResource)(nil)).
			Where("id = ?", id).
			Exec(ctx)
		if err != nil {
			return err
		}
		_, err = tx.NewDelete().
			Model((*models.Submission)(nil)).
			Where("owner_id = ?", id).
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
		pending = append(pending, cacheUserScoresKey(sub.ProblemId, id))
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
			Where("id = ?", subId).
			Scan(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewUpdate().
			Model((*models.Job)(nil)).
			Set("status = ?", models.StatusSkipped).
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

		res, err := tx.NewUpdate().
			Model(job).
			WherePK().
			Where("status != ?", models.StatusSkipped).
			Exec(ctx)
		if err != nil {
			return err
		}
		cnt, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if cnt == 0 {
			return types.JobAlreadySkippedError
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

func (self *Postgres) UpdateSubmissionScore(
	ctx context.Context,
	subId int64,
	score int64,
) error {
	var sub models.Submission
	res, err := self.db.NewUpdate().
		Model(&sub).
		Set("score = ?", score).
		Where("id = ?", subId).
		Returning("owner_id", "problem_id").
		Exec(ctx)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	pending := []string{
		cacheUserScoresKey(sub.ProblemId, sub.OwnerId),
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

func (self *Postgres) CancelSubmission(
	ctx context.Context,
	subId int64,
	ownerId int64,
) ([]string, error) {
	var sub models.Submission
	err := self.db.NewSelect().
		Model(&sub).
		Where("id = ?", subId).
		Where("owner_id = ?", ownerId).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	// Get running jobs before marking them as skipped
	var runningJobs []models.Job
	err = self.db.NewSelect().
		Model(&runningJobs).
		Where("sub_id = ?", subId).
		Where("status = ?", models.StatusRunning).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	runningJobIds := make([]string, len(runningJobs))
	for i, job := range runningJobs {
		runningJobIds[i] = job.Id()
	}

	_, err = self.db.NewUpdate().
		Model((*models.Job)(nil)).
		Set("status = ?", models.StatusSkipped).
		Where("sub_id = ?", subId).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Where("status = ?", models.StatusBlocked).
				WhereOr("status = ?", models.StatusPending)
		}).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	pending := []string{
		cacheUsersKey(sub.ProblemId),
		cacheScoresKey(sub.ProblemId),
		cacheScoresWStagesKey(sub.ProblemId),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return nil, err
		}
	}

	return runningJobIds, nil
}

func (self *Postgres) TimeoutJobById(
	ctx context.Context,
	jobId string,
) error {
	// Parse job ID: format is "subId-stageName-testName"
	parts := strings.SplitN(jobId, "-", 3)
	if len(parts) != 3 {
		return types.WrongFormatError
	}

	subId, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return err
	}
	stageName := parts[1]
	testName := parts[2]

	job := models.Job{
		SubId:     subId,
		StageName: stageName,
		TestName:  testName,
		Status:    models.StatusMissing,
	}

	if err := self.UpdateJob(ctx, &job); err != nil {
		return err
	}

	return self.SkipJobs(ctx, subId)
}

func (self *Postgres) GetUserResource(
	ctx context.Context,
	id int64,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().
		Model(&ret).
		Where("id = ?", id).
		Scan(ctx)
	return ret, err
}

func (self *Postgres) GetUserPass(
	ctx context.Context,
	username string,
	passhash string,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().
		Model(&ret).
		Where("username = ?", username).
		Where("plain_cred.passhash = ?", passhash).
		Relation("PlainCred").
		Scan(ctx)
	return ret, err
}

func (self *Postgres) InsertUserResources(
	ctx context.Context,
	users []models.UserResource,
	creds []models.PlainCred,
) error {
	for i, newUser := range users {
		err := self.db.RunInTx(ctx, nil,
			func(ctx context.Context, tx bun.Tx) error {
				var oldUser models.UserResource
				err := tx.NewSelect().
					Model(&oldUser).
					Where("username = ?", newUser.Username).
					Scan(ctx)
				if err == sql.ErrNoRows {
					// Insert New User
					_, err := tx.NewInsert().
						Model(&newUser).
						ExcludeColumn("id").
						Exec(ctx)
					if err != nil {
						return err
					}
					if creds != nil {
						creds[i].ResourceId = newUser.Id
						_, err = tx.NewInsert().
							Model(&creds[i]).
							On("CONFLICT (resource_id) DO UPDATE").
							Set("passhash = EXCLUDED.passhash").
							Exec(ctx)
						if err != nil {
							return err
						}
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
					Where("username = ?", newUser.Username).
					Exec(ctx)
				if err != nil {
					return err
				}

				if creds != nil {
					creds[i].ResourceId = mergedUser.Id
					_, err = tx.NewInsert().
						Model(&creds[i]).
						On("CONFLICT (resource_id) DO UPDATE").
						Set("passhash = EXCLUDED.passhash").
						Exec(ctx)
					if err != nil {
						return err
					}
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
	id int64,
) (bool, error) {
	return self.db.NewSelect().
		Model((*models.UserResource)(nil)).
		Where("id = ?", id).
		Exists(ctx)
}

func (self *Postgres) ExistUserResourceBySourceId(
	ctx context.Context,
	source models.LoginSourceType,
	sourceId string,
) (bool, error) {
	return self.db.NewSelect().
		Model((*models.UserResource)(nil)).
		Where("source = ?", source).
		Where("source_id = ?", sourceId).
		Exists(ctx)
}

func (self *Postgres) GetUserResourceBySourceId(
	ctx context.Context,
	source models.LoginSourceType,
	sourceId string,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().
		Model(&ret).
		Where("source = ?", source).
		Where("source_id = ?", sourceId).
		Scan(ctx)
	return ret, err
}

func (self *Postgres) UpsertUser(
	ctx context.Context,
	user models.UserResource,
) error {
	_, err := self.db.NewInsert().
		Model(&user).
		On("CONFLICT (id) DO UPDATE").
		Set("args = EXCLUDED.args").
		Set("is_admin = EXCLUDED.is_admin").
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
	id int64,
) (models.Session, error) {
	ret := models.Session{Id: id}
	err := self.db.NewSelect().Model(&ret).WherePK().Scan(ctx)
	return ret, err
}

func (self *Postgres) UpsertSession(
	ctx context.Context,
	session *models.Session,
) error {
	_, err := self.db.NewInsert().
		Model(session).
		On("CONFLICT (id) DO UPDATE").
		Set("id = EXCLUDED.id").
		Set("refresh_token = EXCLUDED.refresh_token").
		Set("is_valid = EXCLUDED.is_valid").
		Exec(ctx)
	return err
}

func (self *Postgres) UpdateRefreshToken(
	ctx context.Context,
	id int64,
) (models.Session, error) {
	ret := models.Session{
		Id: id,
	}
	err := self.db.NewSelect().
		Model(&ret).
		WherePK().
		Scan(ctx)
	if err != nil {
		return models.Session{}, err
	}
	if err := ret.RotateRefreshToken(); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to rotate refresh token",
				zap.Error(err))
		return models.Session{}, err
	}
	err = self.db.NewUpdate().
		Model((*models.Session)(nil)).
		Where("id = ?", id).
		Set("refresh_token = ?", ret.RefreshToken).
		Returning("*").Scan(ctx, &ret)
	return ret, nil
}

func (self *Postgres) InvalidateSession(
	ctx context.Context,
	id int64,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Session)(nil)).
		Where("id = ?", id).
		Set("is_valid = ?", false).
		Exec(ctx)
	return err
}

func (self *Postgres) DeleteUser(
	ctx context.Context,
	id int64,
) error {
	_, err := self.db.NewDelete().
		Model((*models.UserResource)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (self *Postgres) UpdateUserArgsById(
	ctx context.Context,
	user models.UserResource,
) error {
	return self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var oldUser models.UserResource
		err := tx.NewSelect().
			Model(&oldUser).
			Where("id = ?", user.Id).
			Scan(ctx)
		if err != nil {
			return err
		}

		b, err := utils.JSONMerge(ctx, oldUser, user)
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
			Column("args").
			Model(&mergedUser).
			Where("id = ?", user.Id).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

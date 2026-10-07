package implements

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	namespaceId int64,
	problemId int64,
) string {
	return fmt.Sprintf("problem-id-%d-%d", namespaceId, problemId)
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
	namespaceId int64,
	problemId int64,
) (models.Problem, error) {
	var ret models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&ret).
		Where("problem_id = ?", problemId).
		Where("namespace_id = ?", namespaceId).
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
	namespaceId int64,
	problemId int64,
) (models.Problem, error) {
	if self.cache != nil {
		var problem models.Problem
		err := self.cache.Once(&cache.Item{
			Key:   cacheProblemIdKey(namespaceId, problemId),
			Value: &problem,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getProblemById(ctx, namespaceId, problemId)
			},
		})
		return problem, err
	} else {
		return self.getProblemById(ctx, namespaceId, problemId)
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
	var problemId int64
	var namespaceId int64
	if isEnabled {
		err := self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				var problem models.ProblemMarshaled
				err := tx.NewSelect().
					Column("sha1", "problem_id", "namespace_id", "is_enabled").
					Model(&problem).
					Where("sha1 LIKE ?", sha1Prefix+"%").
					Scan(ctx)
				if err != nil {
					return err
				}
				sha1 = problem.SHA1
				problemId = problem.ProblemId
				namespaceId = problem.NamespaceId
				_, err = tx.NewUpdate().
					Model((*models.ProblemMarshaled)(nil)).
					Set("is_enabled = CASE WHEN sha1 = ? THEN TRUE ELSE FALSE END", sha1).
					Where("problem_id = ?", problem.ProblemId).
					Where("namespace_id = ?", problem.NamespaceId).
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
			Returning("sha1", "problem_id", "namespace_id").
			Exec(ctx)
		if err != nil {
			return err
		}
		sha1 = problem.SHA1
		problemId = problem.ProblemId
		namespaceId = problem.NamespaceId
	}
	if self.cache != nil {
		// Invalidate problem caches
		pending := []string{
			cacheProblemSHA1Key(sha1),
			cacheProblemEnabledKey(),
			cacheProblemIdKey(namespaceId, problemId),
			// Invalidate score-related caches for this problem
			// This is necessary when ScoreOutputType changes (e.g., latest -> highest)
			cacheScoresKey(namespaceId, problemId),
			cacheScoresWStagesKey(namespaceId, problemId),
			cacheUsersKey(namespaceId, problemId),
			cacheNamespaceScoreboardKey(namespaceId),
		}

		// Invalidate per-user score caches
		users, err := self.getSubmittedUsers(ctx, namespaceId, problemId)
		if err != nil {
			return err
		}
		for _, userId := range users {
			pending = append(pending,
				cacheUserScoresKey(namespaceId, problemId, userId),
				cacheUserStageScoresKey(namespaceId, problemId, userId),
				cacheUserHighestScoreKey(namespaceId, problemId, userId),
			)
		}

		for _, key := range pending {
			if err := self.cache.Delete(ctx, key); err != nil {
				return err
			}
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
				Returning("sha1, problem_id, namespace_id").
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
			cacheProblemIdKey(problem.NamespaceId, problem.ProblemId)); err != nil {
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
				Returning("sha1, tool_id").
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

// Score cache keys are namespaced because problem_id is only unique within a
// namespace.

func cacheScoresKey(
	namespaceId int64,
	problemId int64,
) string {
	return fmt.Sprintf("scores-%d-%d", namespaceId, problemId)
}

func cacheNamespaceScoreboardKey(
	namespaceId int64,
) string {
	return "scoreboard-" + strconv.FormatInt(namespaceId, 10)
}

func cacheScoresWStagesKey(
	namespaceId int64,
	problemId int64,
) string {
	return fmt.Sprintf("scores-w-stages-%d-%d", namespaceId, problemId)
}

func cacheUserScoresKey(
	namespaceId int64,
	problemId int64,
	id int64,
) string {
	return fmt.Sprintf("user-scores-%d-%d-%d", namespaceId, problemId, id)
}

func cacheSubmissionScoreKey(
	subId int64,
) string {
	return "sub-scores-" + strconv.FormatInt(subId, 10)
}

func cacheUsersKey(
	namespaceId int64,
	problemId int64,
) string {
	return fmt.Sprintf("users-%d-%d", namespaceId, problemId)
}

func cacheUserStageScoresKey(
	namespaceId int64,
	problemId int64,
	id int64,
) string {
	return fmt.Sprintf("user-stage-scores-%d-%d-%d", namespaceId, problemId, id)
}

func cacheUserHighestScoreKey(
	namespaceId int64,
	problemId int64,
	id int64,
) string {
	return fmt.Sprintf("user-highest-score-%d-%d-%d", namespaceId, problemId, id)
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

	if problem.ScoreInputType == models.ScoreInputTypeSelfManaged {
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
	namespaceId int64,
	refId int64,
	id int64,
) (models.UserScoreWithStages, error) {
	// Get the enabled problem to check ScoreOutputType
	enabledProblem, err := self.GetProblemById(ctx, namespaceId, refId)
	if err != nil {
		return models.UserScoreWithStages{}, err
	}

	user, err := self.GetUserResource(ctx, id)
	if err != nil {
		return models.UserScoreWithStages{}, err
	}

	if enabledProblem.ScoreOutputType == models.ScoreOutputTypeHighest {
		return self.getUserHighestScoreWithStages(ctx, namespaceId, refId, id, enabledProblem, user)
	}

	return self.getUserLatestScoreWithStages(ctx, namespaceId, refId, id, user)
}

func (self *Postgres) getUserLatestScoreWithStages(
	ctx context.Context,
	namespaceId int64,
	refId int64,
	id int64,
	user models.UserResource,
) (models.UserScoreWithStages, error) {
	var sub models.Submission
	err := self.db.NewSelect().
		Model(&sub).
		Column("id", "owner_id", "problem_id", "problem_sha1", "created_at", "score").
		Where("problem_id = ?", refId).
		Where("namespace_id = ?", namespaceId).
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

	var scoreResult models.Score
	var isFinal bool

	if problem.ScoreInputType == models.ScoreInputTypeSelfManaged {
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

func (self *Postgres) getUserHighestScoreWithStages(
	ctx context.Context,
	namespaceId int64,
	refId int64,
	id int64,
	_ models.Problem,
	user models.UserResource,
) (models.UserScoreWithStages, error) {
	// Try to get cached highest score
	var cachedHighest models.CachedHighestScore
	var hasCached bool
	var lastCheckTimestamp int64

	if self.cache != nil {
		err := self.cache.Get(ctx, cacheUserHighestScoreKey(namespaceId, refId, id), &cachedHighest)
		if err == nil {
			hasCached = true
			lastCheckTimestamp = cachedHighest.LastCheckTimestamp
		}
	}

	// Query submissions - either all or only new ones since last check
	var subs []models.Submission
	query := self.db.NewSelect().
		Model(&subs).
		Column("id", "owner_id", "problem_id", "problem_sha1", "created_at", "score").
		Where("problem_id = ?", refId).
		Where("namespace_id = ?", namespaceId).
		Where("owner_id = ?", id).
		Relation("Jobs")

	if hasCached && lastCheckTimestamp > 0 {
		// Only fetch submissions after the last check
		query = query.Where("created_at > ?", lastCheckTimestamp)
	}

	err := query.Scan(ctx)
	if err != nil {
		return models.UserScoreWithStages{}, err
	}

	// If no cached data and no submissions, return error
	if !hasCached && len(subs) == 0 {
		return models.UserScoreWithStages{}, fmt.Errorf("no submissions found")
	}

	// Start with cached values if available
	var bestSub models.Submission
	var bestScore models.Score
	var bestIsFinal bool
	bestScoreValue := int64(-1)

	if hasCached {
		bestScore = cachedHighest.Score
		bestScoreValue = cachedHighest.Score.Score
		bestIsFinal = cachedHighest.IsFinal
		bestSub.CreatedAt = cachedHighest.CreatedAt
	}

	// Track if all new submissions are final and the max created_at
	allNewFinal := true
	var maxCreatedAt int64

	// Check new submissions
	for _, sub := range subs {
		// Track max created_at from new submissions
		if sub.CreatedAt > maxCreatedAt {
			maxCreatedAt = sub.CreatedAt
		}

		problem, err := self.GetProblem(ctx, sub.ProblemSHA1)
		if err != nil {
			return models.UserScoreWithStages{}, err
		}

		var scoreResult models.Score
		var isFinal bool

		if problem.ScoreInputType == models.ScoreInputTypeSelfManaged {
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

		if !isFinal {
			allNewFinal = false
		}

		if scoreResult.Score > bestScoreValue {
			bestScoreValue = scoreResult.Score
			bestSub = sub
			bestScore = scoreResult
			bestIsFinal = isFinal
		}
	}

	result := models.UserScoreWithStages{
		Id:        id,
		Username:  user.Username,
		Score:     bestScore,
		CreatedAt: bestSub.CreatedAt,
		IsFinal:   bestIsFinal,
	}

	// Update cache only if all new submissions are final
	// Use max created_at as timestamp to ensure we don't miss any submissions
	if self.cache != nil && allNewFinal && len(subs) > 0 {
		newTimestamp := maxCreatedAt
		if lastCheckTimestamp > newTimestamp {
			newTimestamp = lastCheckTimestamp
		}
		newCached := models.CachedHighestScore{
			UserScoreWithStages: result,
			LastCheckTimestamp:  newTimestamp,
		}
		_ = self.cache.Set(&cache.Item{
			Key:   cacheUserHighestScoreKey(namespaceId, refId, id),
			Value: &newCached,
			TTL:   time.Hour,
		})
	}

	return result, nil
}

func (self *Postgres) GetUserScoreWithStages(
	ctx context.Context,
	namespaceId int64,
	refId int64,
	id int64,
) (models.UserScoreWithStages, error) {
	if self.cache != nil {
		var ret models.UserScoreWithStages
		err := self.cache.Once(&cache.Item{
			Key:   cacheUserScoresKey(namespaceId, refId, id),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getUserScoreWithStages(ctx, namespaceId, refId, id)
			},
		})
		if ret.IsFinal {
			err = self.cache.Expire(ctx, cacheUserScoresKey(namespaceId, refId, id), time.Hour)
			if err != nil {
				return models.UserScoreWithStages{}, err
			}
		}
		return ret, err
	} else {
		return self.getUserScoreWithStages(ctx, namespaceId, refId, id)
	}
}

func (self *Postgres) getSubmittedUsers(
	ctx context.Context,
	namespaceId int64,
	refId int64,
) ([]int64, error) {
	var subs []models.Submission
	err := self.db.NewSelect().
		Model(&subs).
		DistinctOn("owner_id").
		Column("owner_id").
		Where("problem_id = ?", refId).
		Where("namespace_id = ?", namespaceId).
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
	namespaceId int64,
	refId int64,
) ([]int64, error) {
	if self.cache != nil {
		var ret []int64
		err := self.cache.Once(&cache.Item{
			Key:   cacheUsersKey(namespaceId, refId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getSubmittedUsers(ctx, namespaceId, refId)
			},
		})
		return ret, err
	} else {
		return self.getSubmittedUsers(ctx, namespaceId, refId)
	}
}

func (self *Postgres) getScores(
	ctx context.Context,
	namespaceId int64,
	refId int64,
) ([]models.UserScore, bool, error) {
	users, err := self.GetSubmittedUsers(ctx, namespaceId, refId)
	if err != nil {
		return []models.UserScore{}, false, err
	}

	isFinal := true
	var ret []models.UserScore
	for _, user := range users {
		score, err := self.GetUserScoreWithStages(ctx, namespaceId, refId, user)
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
	namespaceId int64,
	refId int64,
) ([]models.UserScore, error) {
	if self.cache != nil {
		var ret []models.UserScore
		var isFinal bool
		err := self.cache.Once(&cache.Item{
			Key:   cacheScoresKey(namespaceId, refId),
			Value: &ret,
			Do: func(*cache.Item) (ret interface{}, err error) {
				ret, isFinal, err = self.getScores(ctx, namespaceId, refId)
				return
			},
		})
		if isFinal {
			err := self.cache.Expire(ctx, cacheScoresKey(namespaceId, refId), time.Hour)
			if err != nil {
				return []models.UserScore{}, err
			}
		}
		return ret, err
	} else {
		ret, _, err := self.getScores(ctx, namespaceId, refId)
		return ret, err
	}
}

func (self *Postgres) getScoresWithStages(
	ctx context.Context,
	namespaceId int64,
	refId int64,
) ([]models.UserScoreWithStages, bool, error) {
	users, err := self.GetSubmittedUsers(ctx, namespaceId, refId)
	if err != nil {
		return []models.UserScoreWithStages{}, false, err
	}

	isFinal := true
	var ret []models.UserScoreWithStages
	for _, user := range users {
		score, err := self.GetUserScoreWithStages(ctx, namespaceId, refId, user)
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
	namespaceId int64,
	refId int64,
) ([]models.UserScoreWithStages, error) {
	if self.cache != nil {
		var ret []models.UserScoreWithStages
		var isFinal bool
		err := self.cache.Once(&cache.Item{
			Key:   cacheScoresWStagesKey(namespaceId, refId),
			Value: &ret,
			Do: func(*cache.Item) (ret interface{}, err error) {
				ret, isFinal, err = self.getScoresWithStages(ctx, namespaceId, refId)
				return
			},
		})
		if isFinal {
			err := self.cache.Expire(ctx, cacheScoresWStagesKey(namespaceId, refId), time.Hour)
			if err != nil {
				return []models.UserScoreWithStages{}, err
			}
		}
		return ret, err
	} else {
		ret, _, err := self.getScoresWithStages(ctx, namespaceId, refId)
		return ret, err
	}
}

func (self *Postgres) GetScoresWithDeadline(
	ctx context.Context,
	namespaceId int64,
	refId int64,
) ([]models.UserScore, error) {
	var problem models.Problem
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var problemMarshaled models.ProblemMarshaled
		err := tx.NewSelect().
			Model(&problemMarshaled).
			Where("problem_id = ?", refId).
			Where("namespace_id = ?", namespaceId).
			Where("is_enabled = ?", true).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return err
		}

		problem, err = problemMarshaled.Unmarshal()
		return err
	})
	if err != nil {
		return []models.UserScore{}, err
	}

	if problem.ScoreOutputType == models.ScoreOutputTypeHighest {
		return self.getHighestScoresWithDeadline(ctx, namespaceId, refId, problem)
	}
	return self.getLatestScoresWithDeadline(ctx, namespaceId, refId, problem)
}

func (self *Postgres) getLatestScoresWithDeadline(
	ctx context.Context,
	namespaceId int64,
	refId int64,
	problem models.Problem,
) ([]models.UserScore, error) {
	var submissions []models.Submission

	validSubQ := self.db.NewSelect().
		Model((*models.Submission)(nil)).
		DistinctOn("owner_id").
		Column("owner_id", "created_at", "id").
		Where("problem_id = ?", refId).
		Where("namespace_id = ?", namespaceId).
		Order("owner_id ASC", "created_at DESC")

	if problem.Deadline != 0 {
		validSubQ = validSubQ.
			Where("created_at < ?", problem.Deadline)
	}

	err := self.db.NewSelect().
		With("valid_sub", validSubQ).
		Model(&submissions).
		Where("id IN (?)", self.db.NewSelect().
			Table("valid_sub").
			Column("id")).
		Order("owner_id ASC").
		Relation("Jobs").
		Scan(ctx)
	if err != nil {
		return []models.UserScore{}, err
	}

	var ret []models.UserScore
	for _, sub := range submissions {
		var score int64
		if problem.ScoreInputType == models.ScoreInputTypeSelfManaged {
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

func (self *Postgres) getHighestScoresWithDeadline(
	ctx context.Context,
	namespaceId int64,
	refId int64,
	problem models.Problem,
) ([]models.UserScore, error) {
	var submissions []models.Submission

	query := self.db.NewSelect().
		Model(&submissions).
		Where("problem_id = ?", refId).
		Where("namespace_id = ?", namespaceId).
		Relation("Jobs")

	if problem.Deadline != 0 {
		query = query.Where("created_at < ?", problem.Deadline)
	}

	err := query.Scan(ctx)
	if err != nil {
		return []models.UserScore{}, err
	}

	// Group submissions by owner and find highest score for each
	userBestScores := make(map[int64]models.UserScore)
	for _, sub := range submissions {
		var score int64
		if problem.ScoreInputType == models.ScoreInputTypeSelfManaged {
			score = sub.Score
		} else {
			calcScore, _ := utils.CalcScore(problem, sub)
			score = calcScore.Score
		}

		existing, exists := userBestScores[sub.OwnerId]
		if !exists || score > existing.Score {
			user, err := self.GetUserResource(ctx, sub.OwnerId)
			if err != nil {
				return []models.UserScore{}, err
			}

			userBestScores[sub.OwnerId] = models.UserScore{
				Score:     score,
				Id:        sub.OwnerId,
				Username:  user.Username,
				CreatedAt: sub.CreatedAt,
			}
		}
	}

	var ret []models.UserScore
	for _, score := range userBestScores {
		ret = append(ret, score)
	}

	return ret, nil
}

func (self *Postgres) getNamespaceEnabledProblemIds(
	ctx context.Context,
	namespaceId int64,
) ([]int64, error) {
	var rows []models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&rows).
		Column("problem_id").
		Where("is_enabled = ?", true).
		Where("namespace_id = ?", namespaceId).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ProblemId)
	}
	return ids, nil
}

func (self *Postgres) getNamespaceScoreboard(
	ctx context.Context,
	namespaceId int64,
) ([]models.ScoreboardEntry, error) {
	problemIds, err := self.getNamespaceEnabledProblemIds(ctx, namespaceId)
	if err != nil {
		return nil, err
	}

	entries := make(map[int64]*models.ScoreboardEntry)

	// Seed every namespace member so zero-score users appear too.
	members, err := self.GetNamespaceMembers(ctx, namespaceId)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		user, err := self.GetUserResource(ctx, m.UserId)
		if err != nil {
			continue
		}
		entries[m.UserId] = &models.ScoreboardEntry{
			Id:       m.UserId,
			Username: user.Username,
			Problems: make(map[int64]int64),
		}
	}

	for _, pid := range problemIds {
		scores, err := self.GetScoresWithDeadline(ctx, namespaceId, pid)
		if err != nil {
			return nil, err
		}
		for _, s := range scores {
			entry, ok := entries[s.Id]
			if !ok {
				entry = &models.ScoreboardEntry{
					Id:       s.Id,
					Username: s.Username,
					Problems: make(map[int64]int64),
				}
				entries[s.Id] = entry
			}
			entry.Total += s.Score
			entry.Problems[pid] = s.Score
			if s.CreatedAt > entry.CreatedAt {
				entry.CreatedAt = s.CreatedAt
			}
		}
	}

	ret := make([]models.ScoreboardEntry, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, *entry)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Total != ret[j].Total {
			return ret[i].Total > ret[j].Total
		}
		if ret[i].CreatedAt != ret[j].CreatedAt {
			return ret[i].CreatedAt < ret[j].CreatedAt
		}
		return ret[i].Username < ret[j].Username
	})
	return ret, nil
}

func (self *Postgres) GetNamespaceScoreboard(
	ctx context.Context,
	namespaceId int64,
) ([]models.ScoreboardEntry, error) {
	if self.cache != nil {
		var ret []models.ScoreboardEntry
		err := self.cache.Once(&cache.Item{
			Key:   cacheNamespaceScoreboardKey(namespaceId),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getNamespaceScoreboard(ctx, namespaceId)
			},
			TTL: 2 * time.Second,
		})
		return ret, err
	}
	return self.getNamespaceScoreboard(ctx, namespaceId)
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
	namespaceIds []int64,
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
	if len(namespaceIds) > 0 {
		query = query.Where("namespace_id IN (?)", bun.In(namespaceIds))
	}
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
		Returning("problem_id, owner_id, namespace_id").
		Exec(ctx, &sub)
	if err != nil {
		return err
	}

	pending := []string{
		cacheUserScoresKey(sub.NamespaceId, sub.ProblemId, sub.OwnerId),
		cacheUserStageScoresKey(sub.NamespaceId, sub.ProblemId, sub.OwnerId),
		cacheUserHighestScoreKey(sub.NamespaceId, sub.ProblemId, sub.OwnerId),
		cacheUsersKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId),
		cacheNamespaceScoreboardKey(sub.NamespaceId),
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
					SubId:       submission.Id,
					NamespaceId: submission.NamespaceId,
					StageName:   stageName,
					TestName:    testcase.Name,

					// Queued; becomes Blocked once a worker claims the submission.
					Status: models.StatusPending,
				}

				_, err = tx.NewInsert().
					Model(&job).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
		}

		// Enqueue the submission in the same transaction as the submission and
		// its jobs, so a submission can never exist without its queue entry.
		queueItem := models.SubmissionQueueItem{
			SubmissionId: submission.Id,
			UserId:       submission.OwnerId,
			ProblemId:    submission.ProblemId,
			ProblemSHA1:  submission.ProblemSHA1,
			Type:         models.RefTypeProblem,
			Tag:          problem.Tag,
			Status:       models.SubQueued,
			EnqueuedAt:   time.Now().Unix(),
		}
		if _, err = tx.NewInsert().Model(&queueItem).Exec(ctx); err != nil {
			return err
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
		cacheUsersKey(submission.NamespaceId, submission.ProblemId),
		cacheScoresKey(submission.NamespaceId, submission.ProblemId),
		cacheScoresWStagesKey(submission.NamespaceId, submission.ProblemId),
		cacheNamespaceScoreboardKey(submission.NamespaceId),
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
			SubId:       submission.Id,
			NamespaceId: submission.NamespaceId,
			StageName:   "tool",
			TestName:    tool.Testcase.Name,

			// Queued; becomes Blocked once a worker claims the submission.
			Status: models.StatusPending,
		}
		_, err = tx.NewInsert().
			Model(&job).
			Exec(ctx)
		if err != nil {
			return err
		}

		// Enqueue in the same transaction as the submission and its job.
		queueItem := models.SubmissionQueueItem{
			SubmissionId: submission.Id,
			UserId:       submission.OwnerId,
			ProblemId:    submission.ToolId,
			ProblemSHA1:  submission.ToolSHA1,
			Type:         models.RefTypeTool,
			Tag:          tool.Tag,
			Status:       models.SubQueued,
			EnqueuedAt:   time.Now().Unix(),
		}
		if _, err = tx.NewInsert().Model(&queueItem).Exec(ctx); err != nil {
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
			Returning("problem_id, namespace_id").
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
		pending = append(pending, cacheUserScoresKey(sub.NamespaceId, sub.ProblemId, id))
		pending = append(pending, cacheUserHighestScoreKey(sub.NamespaceId, sub.ProblemId, id))
		pending = append(pending, cacheUsersKey(sub.NamespaceId, sub.ProblemId))
		pending = append(pending, cacheScoresKey(sub.NamespaceId, sub.ProblemId))
		pending = append(pending, cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId))
		pending = append(pending, cacheNamespaceScoreboardKey(sub.NamespaceId))
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
			Where("status = ?", models.StatusBlocked).
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
		cacheUsersKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId),
		cacheNamespaceScoreboardKey(sub.NamespaceId),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (self *Postgres) SkipJobWithStages(
	ctx context.Context,
	subId int64,
	stages []string,
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
			Where("stage_name IN (?)", bun.In(stages)).
			Where("status = ?", models.StatusBlocked).
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
		cacheUsersKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId),
		cacheNamespaceScoreboardKey(sub.NamespaceId),
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
	from ...models.JobStatus,
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
			Where("status IN (?)", bun.In(from)).
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
		cacheUserScoresKey(sub.NamespaceId, sub.ProblemId, sub.OwnerId),
		cacheUsersKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId),
		cacheNamespaceScoreboardKey(sub.NamespaceId),
	}

	// The status change is already committed - failing here would make the
	// caller believe it never happened (e.g. marshal answering 503 for a job
	// that is now Running, which nobody then runs or reports).
	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			tracing.Logger.Ctx(ctx).
				Warn("failed to invalidate cache after job update",
					zap.String("job.id", job.Id()),
					zap.String("key", key),
					zap.Error(err))
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
		Returning("owner_id, problem_id, problem_sha1, namespace_id").
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

	problem, err := self.GetProblem(ctx, sub.ProblemSHA1)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get problem",
				zap.String("submission", fmt.Sprintf("%+v", sub)),
				zap.String("problem.sha1", sub.ProblemSHA1),
				zap.Error(err))
		return err
	}
	if problem.ScoreInputType != models.ScoreInputTypeSelfManaged {
		return types.ScoreInputTypeNotSelfManaged
	}

	pending := []string{
		cacheUserScoresKey(sub.NamespaceId, sub.ProblemId, sub.OwnerId),
		cacheUserHighestScoreKey(sub.NamespaceId, sub.ProblemId, sub.OwnerId),
		cacheUsersKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId),
		cacheNamespaceScoreboardKey(sub.NamespaceId),
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
) error {
	var sub models.Submission
	err := self.db.NewSelect().
		Model(&sub).
		Where("id = ?", subId).
		Where("owner_id = ?", ownerId).
		Scan(ctx)
	if err != nil {
		return err
	}

	// Skip not-yet-started jobs. Running jobs are killed by the owning worker
	// when it aborts the submission (signalled via the heartbeat).
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
		return err
	}

	pending := []string{
		cacheUsersKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresKey(sub.NamespaceId, sub.ProblemId),
		cacheScoresWStagesKey(sub.NamespaceId, sub.ProblemId),
		cacheNamespaceScoreboardKey(sub.NamespaceId),
	}

	for _, key := range pending {
		if err := self.cache.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
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

	if err := self.UpdateJob(ctx, &job, models.StatusRunning); err != nil {
		return err
	}

	return self.SkipJobs(ctx, subId)
}

func (self *Postgres) GetRunningJobs(
	ctx context.Context,
) ([]models.Job, error) {
	var jobs []models.Job
	err := self.db.NewSelect().
		Model(&jobs).
		Where("status = ?", models.StatusRunning).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (self *Postgres) ClaimNext(
	ctx context.Context,
	workerName string,
	tags []string,
) (models.SubmissionQueueItem, error) {
	var item models.SubmissionQueueItem
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().
			Model(&item).
			Where("status = ?", models.SubQueued).
			Where("tag IN (?)", bun.In(tags)).
			Order("enqueued_at ASC").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			return err
		}

		item.Status = models.SubAssigned
		item.WorkerName = workerName
		item.AssignedAt = time.Now().Unix()
		_, err = tx.NewUpdate().
			Model(&item).
			WherePK().
			Column("status", "worker_name", "assigned_at").
			Exec(ctx)
		return err
	})
	if err != nil {
		return models.SubmissionQueueItem{}, err
	}
	return item, nil
}

func (self *Postgres) CompleteSubmission(
	ctx context.Context,
	submissionId int64,
) error {
	// A completed submission is removed from the queue entirely; the queue only
	// tracks unprocessed (queued) and in-flight (assigned) submissions.
	_, err := self.db.NewDelete().
		Model((*models.SubmissionQueueItem)(nil)).
		Where("submission_id = ?", submissionId).
		Exec(ctx)
	return err
}

func (self *Postgres) RefreshLease(
	ctx context.Context,
	workerName string,
	subIds []int64,
) error {
	if workerName == "" || len(subIds) == 0 {
		return nil
	}
	_, err := self.db.NewUpdate().
		Model((*models.SubmissionQueueItem)(nil)).
		Set("assigned_at = ?", time.Now().Unix()).
		Where("worker_name = ?", workerName).
		Where("status = ?", models.SubAssigned).
		Where("submission_id IN (?)", bun.In(subIds)).
		Exec(ctx)
	return err
}

func (self *Postgres) ReclaimExpired(
	ctx context.Context,
	leaseSeconds int64,
) ([]models.SubmissionQueueItem, error) {
	cutoff := time.Now().Unix() - leaseSeconds
	var reclaimed []models.SubmissionQueueItem
	_, err := self.db.NewUpdate().
		Model((*models.SubmissionQueueItem)(nil)).
		Set("status = ?", models.SubQueued).
		Set("worker_name = NULL").
		Set("assigned_at = NULL").
		Where("status = ?", models.SubAssigned).
		Where("assigned_at < ?", cutoff).
		Returning("*").
		Exec(ctx, &reclaimed)
	if err != nil {
		return nil, err
	}
	return reclaimed, nil
}

func (self *Postgres) SubmissionWorker(
	ctx context.Context,
	submissionId int64,
) (string, error) {
	var item models.SubmissionQueueItem
	err := self.db.NewSelect().
		Model(&item).
		Column("worker_name", "status").
		Where("submission_id = ?", submissionId).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if item.Status != models.SubAssigned {
		return "", nil
	}
	return item.WorkerName, nil
}

func (self *Postgres) BlockSubmissionJobs(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Job)(nil)).
		Set("status = ?", models.StatusBlocked).
		Where("sub_id = ?", subId).
		Where("status = ?", models.StatusPending).
		Exec(ctx)
	return err
}

func (self *Postgres) ResetSubmissionActiveJobs(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Job)(nil)).
		Set("status = ?", models.StatusPending).
		Where("sub_id = ?", subId).
		Where("status IN (?)", bun.In([]models.JobStatus{
			models.StatusRunning,
			models.StatusBlocked,
		})).
		Exec(ctx)
	return err
}

// hydrateUserNamespaces fills Namespaces / AdminIn from user_namespace.
func (self *Postgres) hydrateUserNamespaces(
	ctx context.Context,
	user *models.UserResource,
) error {
	namespaces, adminIn, err := self.GetUserNamespaces(ctx, user.Id)
	if err != nil {
		return err
	}
	// A user with no explicit membership belongs to the default namespace
	// (backward-compatible single-tenant behavior). Global admins of the
	// default namespace mirror their global admin flag.
	if len(namespaces) == 0 {
		namespaces = []int64{models.DefaultNamespaceId}
		if user.IsAdmin {
			adminIn = []int64{models.DefaultNamespaceId}
		}
	}
	user.Namespaces = namespaces
	user.AdminIn = adminIn
	return nil
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
	if err != nil {
		return ret, err
	}
	if err := self.hydrateUserNamespaces(ctx, &ret); err != nil {
		return ret, err
	}
	return ret, nil
}

func (self *Postgres) CreateNamespace(
	ctx context.Context,
	ns *models.Namespace,
) error {
	ns.CreatedAt = time.Now().Unix()
	_, err := self.db.NewInsert().Model(ns).Exec(ctx)
	return err
}

func (self *Postgres) GetNamespaces(
	ctx context.Context,
) ([]models.Namespace, error) {
	var ret []models.Namespace
	err := self.db.NewSelect().Model(&ret).Order("id ASC").Scan(ctx)
	return ret, err
}

func (self *Postgres) GetNamespace(
	ctx context.Context,
	id int64,
) (models.Namespace, error) {
	ns := models.Namespace{Id: id}
	err := self.db.NewSelect().Model(&ns).WherePK().Scan(ctx)
	return ns, err
}

func (self *Postgres) GetNamespaceByName(
	ctx context.Context,
	name string,
) (models.Namespace, error) {
	var ns models.Namespace
	err := self.db.NewSelect().Model(&ns).Where("name = ?", name).Scan(ctx)
	return ns, err
}

func (self *Postgres) UpdateNamespaceState(
	ctx context.Context,
	id int64,
	isEnabled bool,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Namespace)(nil)).
		Set("is_enabled = ?", isEnabled).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (self *Postgres) UpsertNamespaceMembers(
	ctx context.Context,
	namespaceId int64,
	userIds []int64,
	isAdmin bool,
) error {
	if len(userIds) == 0 {
		return nil
	}
	members := make([]models.UserNamespace, len(userIds))
	for i, uid := range userIds {
		members[i] = models.UserNamespace{
			UserId:      uid,
			NamespaceId: namespaceId,
			IsAdmin:     isAdmin,
		}
	}
	_, err := self.db.NewInsert().
		Model(&members).
		On("CONFLICT (user_id, namespace_id) DO UPDATE").
		Set("is_admin = EXCLUDED.is_admin").
		Exec(ctx)
	return err
}

func (self *Postgres) RemoveNamespaceMembers(
	ctx context.Context,
	namespaceId int64,
	userIds []int64,
) error {
	if len(userIds) == 0 {
		return nil
	}
	_, err := self.db.NewDelete().
		Model((*models.UserNamespace)(nil)).
		Where("namespace_id = ?", namespaceId).
		Where("user_id IN (?)", bun.In(userIds)).
		Exec(ctx)
	return err
}

func (self *Postgres) GetNamespaceMembers(
	ctx context.Context,
	namespaceId int64,
) ([]models.UserNamespace, error) {
	var ret []models.UserNamespace
	err := self.db.NewSelect().
		Model(&ret).
		Where("namespace_id = ?", namespaceId).
		Scan(ctx)
	return ret, err
}

func (self *Postgres) GetUserNamespaces(
	ctx context.Context,
	userId int64,
) ([]int64, []int64, error) {
	var memberships []models.UserNamespace
	err := self.db.NewSelect().
		Model(&memberships).
		Where("user_id = ?", userId).
		Scan(ctx)
	if err != nil {
		return nil, nil, err
	}
	var namespaces, adminIn []int64
	for _, m := range memberships {
		namespaces = append(namespaces, m.NamespaceId)
		if m.IsAdmin {
			adminIn = append(adminIn, m.NamespaceId)
		}
	}
	return namespaces, adminIn, nil
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
	if err != nil {
		return ret, err
	}
	if err := self.hydrateUserNamespaces(ctx, &ret); err != nil {
		return ret, err
	}
	return ret, nil
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
				if errors.Is(err, sql.ErrNoRows) {
					// Insert New User
					_, err := tx.NewInsert().
						Model(&newUser).
						ExcludeColumn("id").
						Returning("id").
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
				} else {
					if err != nil {
						return err
					}
					newUser.Id = oldUser.Id
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
					Set("is_admin = ?", mergedUser.IsAdmin).
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
	if err != nil {
		return ret, err
	}
	if err := self.hydrateUserNamespaces(ctx, &ret); err != nil {
		return ret, err
	}
	return ret, nil
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
		Returning("*").
		Scan(ctx, &ret)
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

// -- TokenStorage --
//
// Postgres is the authoritative store for static bearer tokens; when the
// cache is backed by Redis (or a local cache in --single-node mode), reads
// go through self.cache the same way problem/tool/score lookups do.

func tokenSHA256(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}

func cacheTokenSHA256Key(sha256Hex string) string {
	return "token:" + sha256Hex
}

func (self *Postgres) getToken(
	ctx context.Context,
	sha256Hex string,
) (models.Token, error) {
	var token models.Token
	err := self.db.NewSelect().
		Model(&token).
		Where("sha256 = ?", sha256Hex).
		Scan(ctx)
	return token, err
}

func (self *Postgres) GetToken(
	ctx context.Context,
	rawToken string,
) (models.Token, error) {
	sha256Hex := tokenSHA256(rawToken)

	if self.cache != nil {
		var ret models.Token
		err := self.cache.Once(&cache.Item{
			Key:   cacheTokenSHA256Key(sha256Hex),
			Value: &ret,
			Do: func(*cache.Item) (interface{}, error) {
				return self.getToken(ctx, sha256Hex)
			},
		})
		return ret, err
	}
	return self.getToken(ctx, sha256Hex)
}

func (self *Postgres) GetTokens(
	ctx context.Context,
) ([]models.Token, error) {
	var tokens []models.Token
	err := self.db.NewSelect().
		Model(&tokens).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (self *Postgres) SetAdminToken(
	ctx context.Context,
	rawToken string,
) error {
	sha256Hex := tokenSHA256(rawToken)
	token := models.Token{
		SHA256:      sha256Hex,
		Description: "bootstrap admin token (--admin-token)",
		Privs:       models.NewAdminPrivs(0),
		IsEnabled:   true,
		CreatedAt:   time.Now().Unix(),
	}

	_, err := self.db.NewInsert().
		Model(&token).
		On("CONFLICT (sha256) DO UPDATE").
		Set("description = EXCLUDED.description").
		Set("privs = EXCLUDED.privs").
		Set("is_enabled = EXCLUDED.is_enabled").
		Exec(ctx)
	if err != nil {
		return err
	}

	if self.cache != nil {
		return self.cache.Delete(ctx, cacheTokenSHA256Key(sha256Hex))
	}
	return nil
}

func (self *Postgres) GenerateTokenWithoutExpire(
	ctx context.Context,
	privs models.Privileges,
	userId int64,
	description string,
) (string, error) {
	rawToken, err := utils.RandomString(48)
	if err != nil {
		return "", err
	}

	token := models.Token{
		SHA256:      tokenSHA256(rawToken),
		UserId:      userId,
		Description: description,
		Privs:       privs,
		IsEnabled:   true,
		CreatedAt:   time.Now().Unix(),
	}
	if _, err := self.db.NewInsert().Model(&token).Exec(ctx); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (self *Postgres) setTokenState(
	ctx context.Context,
	sha256Hex string,
	isEnabled bool,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.Token)(nil)).
		Set("is_enabled = ?", isEnabled).
		Where("sha256 = ?", sha256Hex).
		Exec(ctx)
	if err != nil {
		return err
	}

	if self.cache != nil {
		return self.cache.Delete(ctx, cacheTokenSHA256Key(sha256Hex))
	}
	return nil
}

func (self *Postgres) DisableToken(
	ctx context.Context,
	sha256Hex string,
) error {
	return self.setTokenState(ctx, sha256Hex, false)
}

func (self *Postgres) EnableToken(
	ctx context.Context,
	sha256Hex string,
) error {
	return self.setTokenState(ctx, sha256Hex, true)
}

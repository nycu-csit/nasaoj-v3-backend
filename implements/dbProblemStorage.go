package implements

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/go-redis/cache/v9"
	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

type DBProblemStorage struct {
	db    *bun.DB
	cache *cache.Cache
}

func NewDBProblemStorage(
	db *bun.DB,
	opt *cache.Options,
) *DBProblemStorage {
	var cash *cache.Cache = nil
	if opt != nil {
		cash = cache.New(opt)
	}
	return &DBProblemStorage{
		db:    db,
		cache: cash,
	}
}

func (self *DBProblemStorage) problemSHA1Key(
	sha1 string,
) string {
	return "problem-" + sha1
}

func (self *DBProblemStorage) problemIdKey(
	problemId int64,
) string {
	return "problem-id-" + strconv.FormatInt(problemId, 10)
}

func (self *DBProblemStorage) problemEnabledKey() string {
	return "problem-enabled"
}

func (self *DBProblemStorage) getProblem(
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

func (self *DBProblemStorage) GetProblem(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	if self.cache != nil {
		var ret models.Problem
		err := self.cache.Once(&cache.Item{
			Key:   self.problemSHA1Key(sha1),
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

func (self *DBProblemStorage) getProblemById(
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

func (self *DBProblemStorage) GetProblemById(
	ctx context.Context,
	problemId int64,
) (models.Problem, error) {
	if self.cache != nil {
		var problem models.Problem
		err := self.cache.Once(&cache.Item{
			Key:   self.problemIdKey(problemId),
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

func (self *DBProblemStorage) InsertProblem(
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

func (self *DBProblemStorage) getProblemsEnabled(
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

func (self *DBProblemStorage) GetProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	var problemsEnabled []string
	if self.cache != nil {
		err := self.cache.Once(&cache.Item{
			Key:   self.problemEnabledKey(),
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
func (self *DBProblemStorage) GetAllProblems(
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

func (self *DBProblemStorage) UpdateProblemState(
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
		return self.cache.Delete(ctx, self.problemEnabledKey())
	} else {
		return nil
	}
}

func (self *DBProblemStorage) GetProblemScores(
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

func (self *DBProblemStorage) DeleteProblemAndSubmissions(
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
			self.problemIdKey(problem.ProblemId)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			self.problemSHA1Key(sha1)); err != nil {
			return err
		}
	}
	return nil
}

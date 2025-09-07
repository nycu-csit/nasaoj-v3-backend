package implements

import (
	"context"
	"text/template"

	"github.com/go-redis/cache/v9"
	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

type DBContentFileStorage struct {
	db    *bun.DB
	cache *cache.Cache
}

func NewDBContentFileStorage(
	db *bun.DB,
	opt *cache.Options,
) *DBContentFileStorage {
	var cash *cache.Cache = nil
	if opt != nil {
		cash = cache.New(opt)
	}
	return &DBContentFileStorage{
		db:    db,
		cache: cash,
	}
}

func (self *DBContentFileStorage) contentFileSHA1Key(
	sha1 string,
) string {
	return "content-file-" + sha1
}

func (self *DBContentFileStorage) getContentFile(
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

func (self *DBContentFileStorage) GetContentFile(
	ctx context.Context,
	sha1 string,
) (models.ContentFile, error) {
	if self.cache != nil {
		var ret models.ContentFile
		err := self.cache.Once(&cache.Item{
			Key:   self.contentFileSHA1Key(sha1),
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

func (self *DBContentFileStorage) GetContentFiles(
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
func (self *DBContentFileStorage) GetAllContentFiles(
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

func (self *DBContentFileStorage) InsertContentFiles(
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

func (self *DBContentFileStorage) HydrateContentFiles(
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

func (self *DBContentFileStorage) HydrateProblem(
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

func (self *DBContentFileStorage) HydrateTool(
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

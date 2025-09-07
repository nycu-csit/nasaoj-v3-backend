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

type DBToolStorage struct {
	db    *bun.DB
	cache *cache.Cache
}

func NewDBToolStorage(
	db *bun.DB,
	opt *cache.Options,
) *DBToolStorage {
	var cash *cache.Cache = nil
	if opt != nil {
		cash = cache.New(opt)
	}
	return &DBToolStorage{
		db:    db,
		cache: cash,
	}
}

func (self *DBToolStorage) toolSHA1Key(
	sha1 string,
) string {
	return "tool-" + sha1
}

func (self *DBToolStorage) toolIdKey(
	toolId int64,
) string {
	return "tool-id-" + strconv.FormatInt(toolId, 10)
}

func (self *DBToolStorage) toolEnabledKey() string {
	return "tool-enabled"
}

func (self *DBToolStorage) getTool(
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

func (self *DBToolStorage) GetTool(
	ctx context.Context,
	sha1 string,
) (models.Tool, error) {
	if self.cache != nil {
		var ret models.Tool
		err := self.cache.Once(&cache.Item{
			Key:   self.toolSHA1Key(sha1),
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

func (self *DBToolStorage) getToolById(
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

func (self *DBToolStorage) GetToolById(
	ctx context.Context,
	toolId int64,
) (models.Tool, error) {
	if self.cache != nil {
		var ret models.Tool
		err := self.cache.Once(&cache.Item{
			Key:   self.toolIdKey(toolId),
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

func (self *DBToolStorage) InsertTool(
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

func (self *DBToolStorage) getToolsEnabled(
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

func (self *DBToolStorage) GetTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	var toolsEnabled []string
	if self.cache != nil {
		err := self.cache.Once(&cache.Item{
			Key:   self.toolEnabledKey(),
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
func (self *DBToolStorage) GetAllTools(
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

func (self *DBToolStorage) UpdateToolState(
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
		return self.cache.Delete(ctx, self.toolEnabledKey())
	} else {
		return nil
	}
}

func (self *DBToolStorage) DeleteToolAndSubmissions(
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
			self.toolIdKey(tool.ToolId)); err != nil {
			return err
		}
		if err := self.cache.Delete(ctx,
			self.toolSHA1Key(sha1)); err != nil {
			return err
		}
	}
	return nil
}

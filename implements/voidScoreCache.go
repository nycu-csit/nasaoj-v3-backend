package implements

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type VoidScoreCache struct{}

func NewVoidScoreCache() *VoidScoreCache {
	return &VoidScoreCache{}
}

func (self *VoidScoreCache) Put(
	_ context.Context,
	_ models.Problem,
	_ string,
	_ string,
	_ int64,
) error {
	return nil
}

func (self *VoidScoreCache) Get(
	_ context.Context,
	_ models.Problem,
	_ bool,
) ([]models.UserScore, error) {
	return []models.UserScore{}, nil
}

func (self *VoidScoreCache) Reload(
	_ context.Context,
	_ interfaces.Database,
	_ models.ProblemRepository,
) error {
	return nil
}

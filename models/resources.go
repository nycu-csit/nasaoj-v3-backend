package models

import (
	"maps"

	"github.com/uptrace/bun"
)

// Resources that bind to specific users
type UserResource struct {
	bun.BaseModel `json:"-" yaml:"-" bun:"table:userresource"`

	CsId string            `json:"csId" yaml:"csId" bun:"cs_id,pk"`
	Args map[string]string `json:"args" yaml:"args" bun:"args"`
}

func (self UserResource) Map() map[string]string {
	ret := map[string]string{
		"csId": self.CsId,
	}
	maps.Copy(ret, self.Args)
	return ret
}

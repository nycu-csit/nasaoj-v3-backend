package models

import (
	"maps"
	"strconv"

	"github.com/uptrace/bun"
)

type LoginSourceType = int

const (
	LoginSourcePlain LoginSourceType = iota
	LoginSourceCSSSO
)

// Resources that bind to specific users
type UserResource struct {
	bun.BaseModel `json:"-" yaml:"-" bun:"table:user_resource"`

	Id       int64             `json:"id" yaml:"id" bun:"id,pk,autoincrement"`
	Username string            `json:"username" yaml:"username" bun:"username,unique"`
	Args     map[string]string `json:"args" yaml:"args" bun:"args"`
	IsAdmin  bool              `json:"isAdmin" yaml:"isAdmin" bun:"is_admin"`

	Source   LoginSourceType `json:"source" yaml:"source" bun:"source"`
	SourceId string          `json:"sourceId" yaml:"sourceId" bun:"source_id"`

	PlainCred *PlainCred `json:"-" yaml:"-" bun:"rel:has-one,join:id=resource_id"`
}

func (self UserResource) Map() map[string]string {
	ret := map[string]string{
		"userId":   strconv.FormatInt(self.Id, 10),
		"username": self.Username,
		"source":   strconv.FormatInt(int64(self.Source), 10),
		"sourceId": self.SourceId,
	}
	maps.Copy(ret, self.Args)
	return ret
}

type PlainCred struct {
	bun.BaseModel `bun:"table:plain_cred"`

	ResourceId int64  `bun:"resource_id,pk,autoincrement"`
	Passhash   string `bun:"passhash"`
}

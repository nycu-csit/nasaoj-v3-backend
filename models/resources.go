package models

import (
	"strconv"

	"github.com/uptrace/bun"
)

// The unassigned resource that user can bind to
type Resource struct {
	bun.BaseModel `bun:"table:resource"`
	Id            int64  `bun:"id,pk,autoincrement"`
	Subnet        string `json:"subnet" bun:"subnet,unique"`
}

func (self Resource) User(csid string) UserResource {
	return UserResource{
		CsId:   csid,
		Id:     self.Id,
		Subnet: self.Subnet,
	}
}

// Resources that bind to specific users
type UserResource struct {
	bun.BaseModel `bun:"table:userresource"`
	CsId          string `json:"csId" bun:"csid,pk"`
	Id            int64  `json:"id" bun:"id,unique"`
	Subnet        string `json:"subnet" bun:"subnet,unique"`
}

func (self UserResource) Map() map[string]string {
	return map[string]string{
		"csid":   self.CsId,
		"id":     strconv.FormatInt(self.Id, 10),
		"subnet": self.Subnet,
	}
}

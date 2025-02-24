package models

import "github.com/uptrace/bun"

// The unassigned resource that user can bind to
type Resource struct {
	bun.BaseModel `bun:"table:resource"`
	Id            int64  `bun:"id,pk,autoincrement"`
	Subnet        string `json:"subnet" bun:"subnet"`
}

func (self Resource) User(csid string) UserResource {
	return UserResource{
		CsId:   csid,
		Subnet: self.Subnet,
	}
}

// Resources that bind to specific users
type UserResource struct {
	bun.BaseModel `bun:"table:userresource"`
	CsId          string `json:"csid" bun:"csid,pk"`
	Subnet        string `json:"subnet" bun:"subnet"`
}

func (self UserResource) Map() map[string]string {
	return map[string]string{
		"csid":   self.CsId,
		"subnet": self.Subnet,
	}
}

package models

import (
	"strconv"

	"github.com/uptrace/bun"
)

// The unassigned resource that user can bind to
type Resource struct {
	bun.BaseModel `bun:"table:resource"`
	RoomId        string `bun:"roomid"`
	Id            int64  `bun:"id,pk,autoincrement"`
}

func (self Resource) User(csid string) UserResource {
	return UserResource{
		CsId: csid,
		Id:   self.Id,
	}
}

// Resources that bind to specific users
type UserResource struct {
	bun.BaseModel `bun:"table:userresource"`
	CsId          string `json:"csId" bun:"csid,pk"`
	RoomId        string `json:"roomId" bun:"roomid"`
	Id            int64  `json:"id" bun:"id,unique"`
}

func (self UserResource) Map() map[string]string {
	return map[string]string{
		"csid":   self.CsId,
		"roomId": self.RoomId,
		"id":     strconv.FormatInt(self.Id, 10),
	}
}

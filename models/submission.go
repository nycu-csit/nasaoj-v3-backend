package models

import "github.com/uptrace/bun"

type RefType = int

const (
	RefTypeProblem RefType = 0
	RefTypeTool    RefType = 1
)

type Submission struct {
	bun.BaseModel `bun:"table:submission"`

	Id      int64   `json:"id" bun:"id,pk,autoincrement"`
	RefSHA1 string  `json:"refSHA1" bun:"ref_sha1"`
	RefId   int64   `json:"refId" bun:"ref_id"`
	Type    RefType `json:"type" bun:"type"`

	CsId      string            `json:"csId" bun:"cs_id"`
	CreatedAt int64             `json:"createdAt" bun:"created_at"`
	Args      map[string]string `json:"args" bun:"args"`
	InputArgs map[string]string `json:"inputArgs" bun:"input_args"`

	Score Score `json:",inline" bun:"embed:score_"`

	Jobs []*Job `bun:"rel:has-many,join:id=sub_id"`
}

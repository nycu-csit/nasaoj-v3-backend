package models

import "github.com/uptrace/bun"

type Submission struct {
	bun.BaseModel `bun:"table:submission"`
	Id            int64  `json:"id" bun:"id,pk,autoincrement"`
	ProblemName   string `json:"problemname" bun:"problemname"`

	CsId     string            `json:"csid" bun:"csid"`
	SubmitAt int64             `json:"submitat" bun:"submitat"`
	Args     map[string]string `json:"args" bun:"args"`

	Score int64 `json:"score" bun:"score"`

	Jobs []*Job `bun:"rel:has-many,join:id=subid"`
}

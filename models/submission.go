package models

import "github.com/uptrace/bun"

type UserScore struct {
	Total  int64            `bun:"total"`
	Stages map[string]int64 `bun:"stages"`
}

type Submission struct {
	bun.BaseModel `bun:"table:submission"`
	Id            int64  `json:"id" bun:"id,pk,autoincrement"`
	ProblemId     int64  `json:"problemId" bun:"problemid"`
	ProblemName   string `json:"problemName" bun:"problemname"`

	CsId     string            `json:"csId" bun:"csid"`
	SubmitAt int64             `json:"submitAt" bun:"submitat"`
	Args     map[string]string `json:"args" bun:"args"`

	Score UserScore `json:",inline" bun:"embed:score_"`

	Jobs []*Job `bun:"rel:has-many,join:id=subid"`
}

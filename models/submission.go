package models

import "github.com/uptrace/bun"

type Score struct {
	Score  int64            `json:"score" bun:"score"`
	Stages map[string]int64 `json:"stages" bun:"stages"`
}

type UserScore struct {
	Score `json:",inline"`
	CsId  string `json:"csId"`
}

type Submission struct {
	bun.BaseModel `bun:"table:submission"`
	Id            int64  `json:"id" bun:"id,pk,autoincrement"`
	ProblemId     int64  `json:"problemId" bun:"problemid"`
	ProblemName   string `json:"problemName" bun:"problemname"`

	CsId     string            `json:"csId" bun:"csid"`
	SubmitAt int64             `json:"submitAt" bun:"submitat"`
	Args     map[string]string `json:"args" bun:"args"`

	Score Score `json:",inline" bun:"embed:score_"`

	Jobs []*Job `bun:"rel:has-many,join:id=subid"`
}

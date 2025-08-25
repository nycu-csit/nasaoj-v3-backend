package models

import "github.com/uptrace/bun"

type Submission struct {
	bun.BaseModel `bun:"table:submission"`
	Id            int64  `json:"id" bun:"id,pk,autoincrement"`
	ProblemSHA1   string `json:"problemSHA1" bun:"problem_sha1"`
	ProblemId     int64  `json:"problemId" bun:"problem_id"`
	ProblemName   string `json:"problemName" bun:"problem_name"`

	CsId     string            `json:"csId" bun:"cs_id"`
	SubmitAt int64             `json:"submitAt" bun:"submit_at"`
	Args     map[string]string `json:"args" bun:"args"`

	Score Score `json:",inline" bun:"embed:score_"`

	Jobs []*Job `bun:"rel:has-many,join:id=subid"`
}

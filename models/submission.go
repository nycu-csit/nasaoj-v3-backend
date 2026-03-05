package models

import "github.com/uptrace/bun"

type RefType = int

const (
	RefTypeProblem RefType = 0
	RefTypeTool    RefType = 1
)

type Submission struct {
	bun.BaseModel `bun:"table:submission"`

	Id          int64   `json:"id" bun:"id,pk,autoincrement"`
	ProblemSHA1 string  `json:"problemSHA1" bun:"problem_sha1"`
	ProblemId   int64   `json:"problemId" bun:"problem_id"`
	ToolSHA1    string  `json:"toolSHA1" bun:"tool_sha1"`
	ToolId      int64   `json:"toolId" bun:"tool_id"`
	Type        RefType `json:"type" bun:"type"`

	OwnerId   int64             `json:"ownerId" bun:"owner_id"`
	CreatedAt int64             `json:"createdAt" bun:"created_at"`
	Args      map[string]string `json:"args" bun:"args"`
	InputArgs map[string]string `json:"inputArgs" bun:"input_args"`

	// self-managed score
	Score int64  `json:"score" bun:"score"`
	Jobs  []*Job `bun:"rel:has-many,join:id=sub_id"`
}

package models

type Score struct {
	Score  int64            `json:"score" bun:"score"`
	Stages map[string]int64 `json:"stages" bun:"stages"`
}

type UserScore struct {
	Score    int64  `json:"score"`
	Id       int64  `json:"id"`
	Username string `json:"username"`

	CreatedAt int64 `json:"createdAt"`
	IsFinal   bool  `json:"-"`
}

type UserScoreWithStages struct {
	Score    `json:",inline"`
	Id       int64  `json:"id"`
	Username string `json:"username"`

	CreatedAt int64 `json:"createdAt"`
	IsFinal   bool  `json:"-"`
}

// CachedHighestScore stores the highest score along with the timestamp
// when it was last computed, enabling incremental updates.
type CachedHighestScore struct {
	UserScoreWithStages
	LastCheckTimestamp int64 `json:"lastCheckTimestamp"`
}

// ScoreboardEntry is a user's total across all problems in a namespace.
type ScoreboardEntry struct {
	Id       int64           `json:"id"`
	Username string          `json:"username"`
	Total    int64           `json:"total"`
	Problems map[int64]int64 `json:"problems,omitempty"` // problemId -> score

	CreatedAt int64 `json:"-"` // latest contributing submission, for tie-breaking
	IsFinal   bool  `json:"-"`
}

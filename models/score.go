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

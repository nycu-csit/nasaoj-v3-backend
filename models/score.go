package models

type Score struct {
	Score  int64            `json:"score" bun:"score"`
	Stages map[string]int64 `json:"stages" bun:"stages"`
}

type UserScore struct {
	Score     int64  `json:"score"`
	CsId      string `json:"csId"`
	CreatedAt int64  `json:"createdAt"`
}

type UserScoreWithStages struct {
	Score     `json:",inline"`
	CsId      string `json:"csId"`
	CreatedAt int64  `json:"createdAt"`
}

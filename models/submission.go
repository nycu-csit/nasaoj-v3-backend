package models

type Submission struct {
	ProblemHydrated
	CsId string `json:"csid"`
	Sat  int64  `json:"sat"` // Submit time
}

package models

type Job struct {
	Id          int               `json:"id" bun:"id,pk"`
	TestName    string            `json:"testname" bun:"testname"`
	TestContent string            `json:"testcontent" bun:"-"`
	Output      string            `json:"output" bun:"-"`
	RetCode     int               `json:"retcode" bun:"retcode"`
	Args        map[string]string `json:"args" bun:"args"`
	Eat         int64             `json:"eat" bun:"eat"` // Executed time
}

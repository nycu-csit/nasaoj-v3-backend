package models

import (
	"fmt"
)

type Job struct {
	SubId     int64  `json:"subid" bun:"subid,pk"`
	StageName string `json:"stagename" bun:"stagename,pk"`
	TestName  string `json:"testname" bun:"testname,pk"`

	TestContent string `json:"testcontent" bun:"-"`
	Output      string `json:"output" bun:"-"`
	RetCode     int    `json:"retcode" bun:"retcode"`
	Eat         int64  `json:"eat" bun:"eat"` // Executed time
}

func (self Job) Id() string {
	return fmt.Sprintf("%d-%s-%s", self.SubId, self.StageName, self.TestName)
}

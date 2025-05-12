package models

import (
	"fmt"

	"github.com/uptrace/bun"
)

// Testcase that schedule to worker
type Job struct {
	bun.BaseModel `bun:"table:job"`
	SubId         int64  `json:"subId" bun:"subid,pk"`
	StageName     string `json:"stageName" bun:"stagename,pk"`
	TestName      string `json:"testName" bun:"testname,pk"`

	TestContent string `json:"testContent" bun:"-"`
	Output      string `json:"output" bun:"-"`
	RetCode     int    `json:"retCode" bun:"retcode"`
	ExecAt      int64  `json:"execAt" bun:"execat"`
}

func (self Job) Id() string {
	return fmt.Sprintf("%d-%s-%s", self.SubId, self.StageName, self.TestName)
}

// Jobcnt key for stage
func (self Job) Stage() string {
	return fmt.Sprintf("%d-%s", self.SubId, self.StageName)
}

package models

import (
	"fmt"

	"github.com/uptrace/bun"
)

// Testcase that schedule to worker
type Job struct {
	bun.BaseModel `bun:"table:job"`
	// Job Id
	SubId     int64  `json:"subId" bun:"subid,pk"`
	StageName string `json:"stageName" bun:"stagename,pk"`
	TestName  string `json:"testName" bun:"testname,pk"`

	// Input script
	Script string `json:"script" bun:"script"`

	// Runtime metadata
	FileURL string `json:"fileurl" bun:"-"`
	HardTL  int64  `json:"hardtl" bun:"-"`
	SoftTL  int64  `json:"softtl" bun:"-"`

	// Return values
	Stdout  string `json:"stdout" bun:"-"`
	Stderr  string `json:"stderr" bun:"-"`
	RetCode int    `json:"retCode" bun:"retcode"`
	ExecAt  int64  `json:"execAt" bun:"execat"`
}

type JobWithFile struct {
	Job
	Tarbuf []byte
}

func (self Job) Id() string {
	return fmt.Sprintf("%d-%s-%s", self.SubId, self.StageName, self.TestName)
}

// Jobcnt key for stage
func (self Job) Stage() string {
	return fmt.Sprintf("%d-%s", self.SubId, self.StageName)
}

package models

import (
	"fmt"

	"github.com/uptrace/bun"
)

type JobStatus = int64

const (
	StatusPending JobStatus = 260
	StatusRunning JobStatus = 261
	StatusSuccess JobStatus = 262
	StatusBlocked JobStatus = 263
	StatusFailed  JobStatus = 460
	StatusSkip    JobStatus = 461
	StatusTimeout JobStatus = 462
	StatusMissing JobStatus = 560
)

// Testcase that schedule to worker
type Job struct {
	bun.BaseModel `bun:"table:job"`
	// Job Id
	SubId     int64  `json:"subId" bun:"subid,pk"`
	StageName string `json:"stageName" bun:"stagename,pk"`
	TestName  string `json:"testName" bun:"testname,pk"`

	// Metadata
	Status JobStatus `json:"status" bun:"status"`

	// On Schedule
	Script   string `json:"script" bun:"script"`
	Deadline int64  `json:"deadline" bun:"deadline"`
	FileURL  string `json:"fileurl" bun:"-"`
	HardTL   int64  `json:"hardtl" bun:"-"`
	SoftTL   int64  `json:"softtl" bun:"-"`

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

func (self Job) Filename() string {
	return fmt.Sprintf("%s.tar", self.Id())
}

// Jobcnt key for stage
func (self Job) Stage() string {
	return fmt.Sprintf("%d-%s", self.SubId, self.StageName)
}

func (self Job) StdoutKey() string {
	return fmt.Sprintf("%s-stdout", self.Id())
}

func (self Job) StderrKey() string {
	return fmt.Sprintf("%s-stderr", self.Id())
}

func (self Job) IsDone() bool {
	return self.Status == StatusSuccess ||
		self.Status == StatusFailed ||
		self.Status == StatusSkip ||
		self.Status == StatusTimeout
}

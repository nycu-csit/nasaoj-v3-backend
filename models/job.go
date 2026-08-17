package models

import (
	"fmt"

	"github.com/uptrace/bun"
)

type JobStatus = int64

/*
	Submission-level scheduling: the backend queues a whole submission and a
	worker claims it; the worker resolves stage dependencies and runs the jobs.

	(Generate jobs on submit) StatusPending          // submission Queued
		-> (Worker claims submission) StatusBlocked  // submission Assigned
			-> (Worker marshals job: POST /schedule/job/marshal) StatusRunning
				-> (PUT /schedule/result)
					-> (If RetCode == 0) StatusSuccess
					-> (If RetCode != 0) StatusFailed
					-> (If timeout) StatusTimeout
					-> (If cancelled) StatusCancelled
				-> (Upstream stage failed) StatusSkipped
			-> (Worker died: reclaim resets to StatusPending; orphan -> StatusMissing)
*/

const (
	StatusPending   JobStatus = 260
	StatusRunning   JobStatus = 261
	StatusSuccess   JobStatus = 262
	StatusBlocked   JobStatus = 263
	StatusFailed    JobStatus = 460
	StatusSkipped   JobStatus = 461
	StatusTimeout   JobStatus = 462
	StatusCancelled JobStatus = 463
	StatusMissing   JobStatus = 560
)

func JobStatusName(status JobStatus) string {
	switch status {
	case StatusPending:
		return "Pending"
	case StatusRunning:
		return "Running"
	case StatusSuccess:
		return "Success"
	case StatusBlocked:
		return "Blocked"
	case StatusFailed:
		return "Failed"
	case StatusSkipped:
		return "Skipped"
	case StatusTimeout:
		return "Timeout"
	case StatusCancelled:
		return "Cancelled"
	case StatusMissing:
		return "Missing"
	default:
		return fmt.Sprintf("Unknown(%d)", status)
	}
}

// Testcase that schedule to worker
type Job struct {
	bun.BaseModel `bun:"table:job"`
	// Job Id
	SubId     int64  `json:"subId" bun:"sub_id,pk"`
	StageName string `json:"stageName" bun:"stage_name,pk"`
	TestName  string `json:"testName" bun:"test_name,pk"`

	// Metadata
	NamespaceId int64     `json:"namespaceId" bun:"namespace_id"`
	Status      JobStatus `json:"status" bun:"status"`

	// On Schedule
	Script        string `json:"script" bun:"script"`
	Input         string `json:"input" bun:"input"`
	ResourceGroup string `json:"resourceGroup" bun:"-"`
	Timeout       int64  `json:"timeout" bun:"-"`
	GracePeriod   int64  `json:"gracePeriod" bun:"-"`
	RefSHA1       string `json:"refSHA1" bun:"-"`
	RefFileURL    string `json:"refFileUrl" bun:"-"`
	JobFileURL    string `json:"jobFileUrl" bun:"-"`
	Image         string `json:"image,omitempty" bun:"-"`

	// Artifacts (glob patterns, from Testcase.Artifacts) and where the
	// worker should upload the packed result, presigned by the backend at
	// marshal time. Empty when the testcase declares no artifacts to keep.
	Artifacts         []string `json:"artifacts,omitempty" bun:"-"`
	ArtifactUploadURL string   `json:"artifactUploadUrl,omitempty" bun:"-"`

	// HasArtifacts is set by the worker (from whether any file actually
	// matched Artifacts) and persisted, so callers can tell whether
	// GET /api/artifacts has anything to return without probing object
	// storage themselves.
	HasArtifacts bool `json:"hasArtifacts,omitempty" bun:"has_artifacts"`

	// Where the worker uploads stdout/stderr directly, presigned by the
	// backend at marshal time - the backend never sees the bytes.
	StdoutUploadURL string `json:"stdoutUploadUrl,omitempty" bun:"-"`
	StderrUploadURL string `json:"stderrUploadUrl,omitempty" bun:"-"`

	// Return values. Left empty on the wire once the worker has uploaded
	// them directly via StdoutUploadURL/StderrUploadURL - they exist here
	// only for the worker's own in-process use while running the job.
	Stdout  string `json:"stdout,omitempty" bun:"-"`
	Stderr  string `json:"stderr,omitempty" bun:"-"`
	RetCode int    `json:"retCode" bun:"ret_code"`
	ExecAt  int64  `json:"execAt" bun:"exec_at"`
}

type JobWithFile struct {
	Job
	Tarbuf []byte

	// SeedDirs are ancestor stages' artifact directories (worker-local
	// filesystem paths) to copy into this job's working directory before it
	// runs. Never sent over the wire - populated by the worker itself from
	// the submission's stage DAG.
	SeedDirs []string
}

func (self Job) Id() string {
	return fmt.Sprintf("%d-%s-%s", self.SubId, self.StageName, self.TestName)
}

func (self Job) JobFilename() string {
	return fmt.Sprintf("job-%s.tar", self.Id())
}

func (self Job) RefFilename() string {
	return fmt.Sprintf("ref-%s-%s-%s.tar", self.RefSHA1, self.StageName,
		self.TestName)
}

func (self Job) ArtifactFilename() string {
	return fmt.Sprintf("artifact-%s.tar", self.Id())
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
		self.Status == StatusSkipped ||
		self.Status == StatusTimeout ||
		self.Status == StatusCancelled ||
		self.Status == StatusMissing
}

package types

import "fmt"

var (
	ContextNotExistError    = fmt.Errorf("context not exist")
	FileNotExistError       = fmt.Errorf("file not exist")
	StageNotExistError      = fmt.Errorf("stage not exist")
	TokenNotExistError      = fmt.Errorf("token not exist")
	JobNotExistError        = fmt.Errorf("job not exist")
	ProblemNotExistError    = fmt.Errorf("problem not exist")
	SubmissionNotExistError = fmt.Errorf("submission not exist")

	ProblemContainsCycleError    = fmt.Errorf("problem contains cycle")
	TokenNotEnabledError         = fmt.Errorf("token not enabled")
	ChannelClosedError           = fmt.Errorf("channel closed")
	WrongFormatError             = fmt.Errorf("wrong format")
	HTTPRequestFailedError       = fmt.Errorf("http request failed")
	LockAcquisitionError         = fmt.Errorf("lock acquisition failed")
	ScoreInputTypeNotSelfManaged = fmt.Errorf("score type is not self managed")

	JobAlreadySkippedError = fmt.Errorf("job has already skipped")
)

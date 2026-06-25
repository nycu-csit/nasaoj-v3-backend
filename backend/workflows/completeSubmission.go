package workflows

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// CompleteIfDone removes a submission from the queue once all of its jobs have
// reached a terminal state. It is idempotent (CompleteSubmission is a no-op when
// the row is already gone) and treats a submission with no jobs as done.
func CompleteIfDone(
	ctx context.Context,
	submissions interfaces.SubmissionStorage,
	subQueue interfaces.SubmissionQueueStorage,
	subId int64,
) error {
	sub, err := submissions.GetSubmission(ctx, subId)
	if err != nil {
		return err
	}
	for _, job := range sub.Jobs {
		if !job.IsDone() {
			return nil
		}
	}
	return subQueue.CompleteSubmission(ctx, subId)
}

package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	type submissionQueue struct {
		bun.BaseModel `bun:"table:submission_queue"`

		SubmissionId int64  `bun:"submission_id,pk"`
		UserId       int64  `bun:"user_id"`
		ProblemId    int64  `bun:"problem_id"`
		ProblemSHA1  string `bun:"problem_sha1"`
		Type         int    `bun:"type"`
		Tag          string `bun:"tag"`

		Status     int    `bun:"status"`
		WorkerName string `bun:"worker_name,nullzero"`
		AssignedAt int64  `bun:"assigned_at,nullzero"`
		EnqueuedAt int64  `bun:"enqueued_at"`
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		_, err := db.NewCreateTable().
			Model((*submissionQueue)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create submission_queue table: %w", err)
		}

		_, err = db.NewCreateIndex().
			Model((*submissionQueue)(nil)).
			Index("submission_queue_claim_idx").
			Column("status", "enqueued_at").
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create submission_queue index: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.NewDropTable().
			Model((*submissionQueue)(nil)).
			IfExists().
			Exec(ctx)
		return err
	})
}

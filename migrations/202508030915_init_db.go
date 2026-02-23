package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	type problemMarshaled struct {
		bun.BaseModel `bun:"table:problem"`

		Payload   string `bun:"payload"`
		ProblemId int64  `bun:"problem_id"`

		SHA1       string `bun:"sha1,pk"`
		ImportedAt int64  `bun:"imported_at"`
		IsEnabled  bool   `bun:"is_enabled"`
	}

	type toolMarshaled struct {
		bun.BaseModel `bun:"table:tool"`

		Payload string `bun:"payload"`
		ToolId  int64  `bun:"tool_id"`

		SHA1       string `bun:"sha1,pk"`
		ImportedAt int64  `bun:"imported_at"`
		IsEnabled  bool   `bun:"is_enabled"`
	}

	type contentFile struct {
		bun.BaseModel `bun:"table:content_file"`

		SHA1       string `bun:"sha1,pk"`
		Name       string `bun:"name"`
		Payload    string `bun:"payload"`
		IsTemplate bool   `bun:"is_template"`
	}

	type job struct {
		bun.BaseModel `bun:"table:job"`

		SubId     int64  `bun:"sub_id,pk"`
		StageName string `bun:"stage_name,pk"`
		TestName  string `bun:"test_name,pk"`

		Status int64 `bun:"status"`

		Script   string `bun:"script"`
		Input    string `bun:"input"`
		Deadline int64  `bun:"deadline"`

		RetCode int   `bun:"ret_code"`
		ExecAt  int64 `bun:"exec_at"`
	}

	type submission struct {
		bun.BaseModel `bun:"submission"`

		Id          int64  `bun:"id,pk,autoincrement"`
		ProblemSHA1 string `bun:"problem_sha1"`
		ProblemId   int64  `bun:"problem_id"`
		ToolSHA1    string `bun:"tool_sha1"`
		ToolId      int64  `bun:"tool_id"`
		Type        int    `bun:"type"`

		CsId      string            `bun:"cs_id"`
		CreatedAt int64             `bun:"created_at"`
		Args      map[string]string `bun:"args"`
		InputArgs map[string]string `bun:"input_args"`
	}

	type userResource struct {
		bun.BaseModel `bun:"table:userresource"`

		CsId string            `bun:"cs_id,pk"`
		Args map[string]string `bun:"args"`
	}

	type session struct {
		bun.BaseModel `bun:"table:session"`

		CsId   string   `bun:"cs_id,pk"`
		Groups []string `bun:"groups"`

		RefreshToken string `bun:"refresh_token"`
		IsValid      bool   `bun:"is_valid"`
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")
		return db.ResetModel(ctx,
			(*problemMarshaled)(nil),
			(*toolMarshaled)(nil),
			(*contentFile)(nil),
			(*job)(nil),
			(*submission)(nil),
			(*userResource)(nil),
			(*session)(nil))
	}, nil)
}

package migrations

import (
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/uptrace/bun"
)

// Content file payloads may be binary, so content_file.payload becomes bytea
// and a content file's key becomes the sha1 of its payload's base64 encoding
// (it used to be the sha1 of the raw payload). Existing rows are re-keyed,
// and the content files embedded in every stored problem/tool payload get
// their new sha1 and a base64-encoded payload, matching ContentFile's JSON
// encoding. Problem/tool row sha1s are left as they are - submissions
// reference them.
func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx, `
				ALTER TABLE content_file
				ALTER COLUMN payload TYPE bytea USING convert_to(payload, 'UTF8')
			`)
			if err != nil {
				return fmt.Errorf("failed to convert content_file.payload to bytea: %w", err)
			}

			return rekeyContentFiles(ctx, tx, true)
		})
	}, func(ctx context.Context, db *bun.DB) error {
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if err := rekeyContentFiles(ctx, tx, false); err != nil {
				return err
			}

			// Fails on payloads that are not valid UTF-8, which the old text
			// column cannot hold anyway.
			_, err := tx.ExecContext(ctx, `
				ALTER TABLE content_file
				ALTER COLUMN payload TYPE text USING convert_from(payload, 'UTF8')
			`)
			if err != nil {
				return fmt.Errorf("failed to convert content_file.payload to text: %w", err)
			}
			return nil
		})
	})
}

func contentFileSHA1(payload []byte, base64Keyed bool) string {
	if base64Keyed {
		payload = []byte(base64.StdEncoding.EncodeToString(payload))
	}
	h := sha1.Sum(payload)
	return hex.EncodeToString(h[:])
}

// rekeyContentFiles moves every content file to the sha1 scheme selected by
// base64Keyed and rewrites the content files embedded in problem/tool
// payloads to match: with base64Keyed their payloads become base64 strings,
// without it they are decoded back to raw strings.
func rekeyContentFiles(ctx context.Context, tx bun.Tx, base64Keyed bool) error {
	type contentFile struct {
		bun.BaseModel `bun:"table:content_file"`

		SHA1       string `bun:"sha1,pk"`
		Name       string `bun:"name"`
		Payload    []byte `bun:"payload"`
		IsTemplate bool   `bun:"is_template"`
	}

	var files []contentFile
	if err := tx.NewSelect().Model(&files).Scan(ctx); err != nil {
		return fmt.Errorf("failed to select content files: %w", err)
	}

	// Rewrite the whole table at once rather than updating keys one row at
	// a time, so a new key can never collide with another row's old key.
	newSHA1 := make(map[string]string, len(files))
	for i := range files {
		key := contentFileSHA1(files[i].Payload, base64Keyed)
		newSHA1[files[i].SHA1] = key
		files[i].SHA1 = key
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM content_file`); err != nil {
		return fmt.Errorf("failed to clear content files: %w", err)
	}
	if len(files) > 0 {
		_, err := tx.NewInsert().
			Model(&files).
			On("CONFLICT (sha1) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to re-insert content files: %w", err)
		}
	}

	for _, table := range []string{"problem", "tool"} {
		var rows []struct {
			SHA1    string `bun:"sha1"`
			Payload string `bun:"payload"`
		}
		err := tx.NewSelect().
			Table(table).
			Column("sha1", "payload").
			Scan(ctx, &rows)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to select %s payloads: %w", table, err)
		}

		for _, row := range rows {
			dec := json.NewDecoder(bytes.NewReader([]byte(row.Payload)))
			dec.UseNumber()
			var v any
			if err := dec.Decode(&v); err != nil {
				return fmt.Errorf("failed to decode %s %s: %w", table, row.SHA1, err)
			}
			if err := rewriteContentFiles(v, newSHA1, base64Keyed); err != nil {
				return fmt.Errorf("failed to rewrite %s %s: %w", table, row.SHA1, err)
			}
			b, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to encode %s %s: %w", table, row.SHA1, err)
			}

			_, err = tx.NewUpdate().
				Table(table).
				Set("payload = ?", string(b)).
				Where("sha1 = ?", row.SHA1).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update %s %s: %w", table, row.SHA1, err)
			}
		}
	}

	return nil
}

// rewriteContentFiles walks a decoded problem/tool payload and rewrites every
// embedded content file object (the only objects carrying sha1, payload and
// isTemplate together) in place.
func rewriteContentFiles(v any, newSHA1 map[string]string, base64Keyed bool) error {
	switch v := v.(type) {
	case []any:
		for _, e := range v {
			if err := rewriteContentFiles(e, newSHA1, base64Keyed); err != nil {
				return err
			}
		}
	case map[string]any:
		sha1, isSHA1 := v["sha1"].(string)
		payload, isPayload := v["payload"].(string)
		_, isTemplate := v["isTemplate"]
		if !isSHA1 || !isPayload || !isTemplate {
			for _, e := range v {
				if err := rewriteContentFiles(e, newSHA1, base64Keyed); err != nil {
					return err
				}
			}
			return nil
		}

		var raw []byte
		if base64Keyed {
			raw = []byte(payload)
			v["payload"] = base64.StdEncoding.EncodeToString(raw)
		} else {
			b, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				return fmt.Errorf("content file %s payload: %w", sha1, err)
			}
			raw = b
			v["payload"] = string(raw)
		}

		if key, ok := newSHA1[sha1]; ok {
			v["sha1"] = key
		} else if len(raw) > 0 {
			v["sha1"] = contentFileSHA1(raw, base64Keyed)
		}
	}
	return nil
}

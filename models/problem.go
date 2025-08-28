package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type ProblemIdType = int64
type TagType = string
type Problem struct {
	ProblemId ProblemIdType `json:"problemId"`

	Name        string            `json:"name"`
	Description string            `json:"description"`
	Stages      map[string]Stage  `json:"stages"`
	Args        map[string]string `json:"args"`
	Tag         TagType           `json:"tag"`
	PreScript   Content           `json:"preScript"`
	PostScript  Content           `json:"postScript"`
	Timeout     int64             `json:"timeout"`

	ImportedAt int64 `json:"-"`
	IsEnabled  bool  `json:"-"`
}

type ProblemMarshaled struct {
	bun.BaseModel `bun:"table:problem"`

	SHA1       string        `bun:"sha1,pk"`
	Payload    string        `bun:"payload"`
	ProblemId  ProblemIdType `bun:"problem_id"`
	ImportedAt int64         `bun:"imported_at"`
	IsEnabled  bool          `bun:"is_enabled"`
}

func (self Problem) Marshal() (ProblemMarshaled, error) {
	ret := ProblemMarshaled{
		ProblemId:  self.ProblemId,
		ImportedAt: time.Now().Unix(),
		IsEnabled:  false,
	}

	b, err := json.Marshal(self)
	if err != nil {
		return ret, err
	}
	ret.Payload = string(b[:])
	h := sha1.Sum(b)
	ret.SHA1 = hex.EncodeToString(h[:])
	return ret, nil
}

func (self ProblemMarshaled) Unmarshal() (Problem, error) {
	var ret Problem
	err := json.Unmarshal([]byte(self.Payload), &ret)
	if err != nil {
		return Problem{}, err
	}
	ret.IsEnabled = self.IsEnabled
	ret.ImportedAt = self.ImportedAt
	return ret, nil
}

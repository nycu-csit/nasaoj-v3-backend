package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"

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

	IsTool    bool `json:"isTool"`
	IsEnabled bool `json:"-"`
}

type Tool = Problem

type ProblemMarshaled struct {
	bun.BaseModel `bun:"table:problem"`

	SHA1    string `bun:"sha1,pk"`
	Payload string `bun:"payload"`

	ProblemId ProblemIdType `bun:"problem_id"`
	IsTool    bool          `bun:"is_tool"`
	IsEnabled bool          `bun:"is_enabled"`
}

func (self Problem) Marshal() (ProblemMarshaled, error) {
	ret := ProblemMarshaled{
		ProblemId: self.ProblemId,
		IsTool:    self.IsTool,
		IsEnabled: false,
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
		return ret, err
	}
	ret.IsTool = self.IsTool
	ret.IsEnabled = self.IsEnabled
	return ret, nil
}

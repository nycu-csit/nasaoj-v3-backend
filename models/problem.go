package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type Problem struct {
	ProblemId   int64             `json:"problemId"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	PdfUrl      string            `json:"pdfUrl"`
	Stages      map[string]Stage  `json:"stages"`
	Args        map[string]string `json:"args"`
	Tag         string            `json:"tag"`
	PreScript   Content           `json:"preScript"`
	PostScript  Content           `json:"postScript"`
	Timeout     int64             `json:"timeout"`
	Deadline    int64             `json:"deadline"`

	ImportedAt int64 `json:"-" msgpack:"importedAt"`
	IsEnabled  bool  `json:"-" msgpack:"isEnabled"`
	TotalScore int64 `json:"-" msgpack:"totalScore"`
}

type ProblemMarshaled struct {
	bun.BaseModel `bun:"table:problem"`

	Payload   string `bun:"payload"`
	ProblemId int64  `bun:"problem_id"`

	SHA1       string `bun:"sha1,pk"`
	ImportedAt int64  `bun:"imported_at"`
	IsEnabled  bool   `bun:"is_enabled"`
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
	ret.TotalScore = 0
	for _, stage := range ret.Stages {
		ret.TotalScore += stage.Score
	}
	return ret, nil
}

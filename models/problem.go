package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type ScoreInputTypeType = int

const (
	ScoreInputTypeDefault ScoreInputTypeType = iota
	ScoreInputTypeSelfManaged
)

type ScoreOutputTypeType = int

const (
	ScoreOutputTypeLatest ScoreOutputTypeType = iota
	ScoreOutputTypeHighest
)

type ProblemSpec struct {
	Type string `json:"type" yaml:"type"`
	Link string `json:"link" yaml:"link"`
}

type Problem struct {
	ProblemId       int64               `json:"problemId" yaml:"problemId"`
	Name            string              `json:"name" yaml:"name"`
	Description     string              `json:"description" yaml:"description"`
	Spec            ProblemSpec         `json:"spec" yaml:"spec"`
	Stages          map[string]Stage    `json:"stages" yaml:"stages"`
	Args            map[string]string   `json:"args" yaml:"args"`
	Tag             string              `json:"tag" yaml:"tag"`
	PreScript       Content             `json:"preScript" yaml:"preScript"`
	PostScript      Content             `json:"postScript" yaml:"postScript"`
	Cooldown        int64               `json:"cooldown" yaml:"cooldown"`
	Deadline        int64               `json:"deadline" yaml:"deadline"`
	ScoreInputType  ScoreInputTypeType  `json:"scoreInputType" yaml:"scoreInputType"`
	ScoreOutputType ScoreOutputTypeType `json:"scoreOutputType" yaml:"scoreOutputType"`

	ImportedAt int64 `json:"importedAt" yaml:"importedAt"`
	IsEnabled  bool  `json:"isEnabled" yaml:"isEnabled"`
	TotalScore int64 `json:"totalScore" yaml:"totalScore"`
}

type problemHash struct {
	ProblemId       int64               `json:"problemId"`
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Spec            ProblemSpec         `json:"spec"`
	Stages          map[string]Stage    `json:"stages"`
	Args            map[string]string   `json:"args"`
	Tag             string              `json:"tag"`
	PreScript       Content             `json:"preScript"`
	PostScript      Content             `json:"postScript"`
	Cooldown        int64               `json:"cooldown"`
	Deadline        int64               `json:"deadline"`
	ScoreInputType  ScoreInputTypeType  `json:"scoreInputType"`
	ScoreOutputType ScoreOutputTypeType `json:"scoreOutputType"`
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

	b, err := json.Marshal(problemHash{
		ProblemId:       self.ProblemId,
		Name:            self.Name,
		Description:     self.Description,
		Spec:            self.Spec,
		Stages:          self.Stages,
		Args:            self.Args,
		Tag:             self.Tag,
		PreScript:       self.PreScript,
		PostScript:      self.PostScript,
		Cooldown:        self.Cooldown,
		Deadline:        self.Deadline,
		ScoreInputType:  self.ScoreInputType,
		ScoreOutputType: self.ScoreOutputType,
	})
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

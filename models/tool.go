package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type ToolInputArg struct {
	Name string                  `json:"name"`
	Type types.ToolInputTypeType `json:"type"`
}

type Tool struct {
	ToolId      int64             `json:"toolId"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Testcase    Testcase          `json:"testcase"`
	Args        map[string]string `json:"args"`
	InputArgs   []ToolInputArg    `json:"inputArgs"`
	Tag         string            `json:"tag"`
	PreScript   Content           `json:"preScript"`
	PostScript  Content           `json:"postScript"`
	Cooldown    int64             `json:"cooldown"`

	NamespaceId int64 `json:"namespaceId"`
	ImportedAt  int64 `json:"importedAt"`
	IsEnabled   bool  `json:"isEnabled"`
}

type toolHash struct {
	ToolId      int64             `json:"toolId"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Testcase    Testcase          `json:"testcase"`
	Args        map[string]string `json:"args"`
	InputArgs   []ToolInputArg    `json:"inputArgs"`
	Tag         string            `json:"tag"`
	PreScript   Content           `json:"preScript"`
	PostScript  Content           `json:"postScript"`
	Cooldown    int64             `json:"cooldown"`
}

type ToolMarshaled struct {
	bun.BaseModel `bun:"table:tool"`

	SHA1        string `bun:"sha1,pk"`
	Payload     string `bun:"payload"`
	ToolId      int64  `bun:"tool_id"`
	NamespaceId int64  `bun:"namespace_id"`
	ImportedAt  int64  `bun:"imported_at"`
	IsEnabled   bool   `bun:"is_enabled"`
}

func (self Tool) Marshal() (ToolMarshaled, error) {
	ret := ToolMarshaled{
		ToolId:      self.ToolId,
		NamespaceId: self.NamespaceId,
		ImportedAt:  time.Now().Unix(),
		IsEnabled:   false,
	}

	b, err := json.Marshal(toolHash{
		ToolId:      self.ToolId,
		Name:        self.Name,
		Description: self.Description,
		Testcase:    self.Testcase,
		Args:        self.Args,
		InputArgs:   self.InputArgs,
		Tag:         self.Tag,
		PreScript:   self.PreScript,
		PostScript:  self.PostScript,
		Cooldown:    self.Cooldown,
	})
	if err != nil {
		return ret, err
	}
	ret.Payload = string(b[:])
	ret.SHA1 = refSHA1(b, self.NamespaceId)
	return ret, nil
}

func (self ToolMarshaled) Unmarshal() (Tool, error) {
	var ret Tool
	err := json.Unmarshal([]byte(self.Payload), &ret)
	if err != nil {
		return Tool{}, err
	}
	ret.IsEnabled = self.IsEnabled
	ret.ImportedAt = self.ImportedAt
	ret.NamespaceId = self.NamespaceId
	return ret, nil
}

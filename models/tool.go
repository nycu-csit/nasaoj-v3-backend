package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type Tool struct {
	ToolId int64 `json:"toolId"`

	Name        string            `json:"name"`
	Description string            `json:"description"`
	Testcase    Testcase          `json:"testcase"`
	Args        map[string]string `json:"args"`
	InputArgs   []string          `json:"inputArgs"`
	Tag         string            `json:"tag"`
	PreScript   Content           `json:"preScript"`
	PostScript  Content           `json:"postScript"`
	Timeout     int64             `json:"timeout"`

	ImportedAt int64 `json:"-" msgpack:"importedAt"`
	IsEnabled  bool  `json:"-" msgpack:"isEnabled"`
}

type ToolMarshaled struct {
	bun.BaseModel `bun:"table:tool"`

	SHA1       string `bun:"sha1,pk"`
	Payload    string `bun:"payload"`
	ToolId     int64  `bun:"tool_id"`
	ImportedAt int64  `bun:"imported_at"`
	IsEnabled  bool   `bun:"is_enabled"`
}

func (self Tool) Marshal() (ToolMarshaled, error) {
	ret := ToolMarshaled{
		ToolId:     self.ToolId,
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

func (self ToolMarshaled) Unmarshal() (Tool, error) {
	var ret Tool
	err := json.Unmarshal([]byte(self.Payload), &ret)
	if err != nil {
		return Tool{}, err
	}
	ret.IsEnabled = self.IsEnabled
	ret.ImportedAt = self.ImportedAt
	return ret, nil
}

package models

import (
	"text/template"

	"github.com/uptrace/bun"
)

type ContentFile struct {
	bun.BaseModel `bun:"table:content_file"`

	SHA1       string `json:"sha1" bun:"sha1,pk"`
	Name       string `json:"name" bun:"name"`
	Payload    string `json:"-" bun:"payload" msgpack:"payload"`
	IsTemplate bool   `json:"-" bun:"is_template" msgpack:"isTemplate"`

	Template *template.Template `json:"-" bun:"-" msgpack:"-"`
}

type Content struct {
	Files   []ContentFile `json:"files"`
	Scripts []string      `json:"scripts"`
}

type Testcase struct {
	Content `json:",inline"`

	Name   string            `json:"name"`
	Args   map[string]string `json:"args"`
	SoftTL int64             `json:"softTL"`
	HardTL int64             `json:"hardTL"`
}

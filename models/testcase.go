package models

import (
	"text/template"

	"github.com/uptrace/bun"
)

type ContentFile struct {
	bun.BaseModel `bun:"table:content_file" json:"-" yaml:"-"`

	SHA1       string `json:"sha1" yaml:"sha1" bun:"sha1,pk"`
	Name       string `json:"name" yaml:"name" bun:"name"`
	Payload    string `json:"payload" yaml:"payload,omitempty" bun:"payload"`
	IsTemplate bool   `json:"isTemplate" yaml:"isTemplate,omitempty" bun:"is_template"`

	Template *template.Template `json:"-" yaml:"-" bun:"-"`
}

type Content struct {
	Files   []ContentFile `json:"files" yaml:"files"`
	Scripts []string      `json:"scripts" yaml:"scripts"`
}

type Testcase struct {
	Content `json:",inline" yaml:"content,inline"`

	Name          string            `json:"name" yaml:"name"`
	Args          map[string]string `json:"args" yaml:"args"`
	Timeout       int64             `json:"timeout" yaml:"timeout"`
	ResourceGroup string            `json:"resourceGroup" yaml:"resourceGroup"`
	Image         string            `json:"image,omitempty" yaml:"image,omitempty"`
}

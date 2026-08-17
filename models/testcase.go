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
	GracePeriod   int64             `json:"gracePeriod" yaml:"gracePeriod"`
	ResourceGroup string            `json:"resourceGroup" yaml:"resourceGroup"`
	Image         string            `json:"image,omitempty" yaml:"image,omitempty"`

	// Artifacts lists glob patterns (matched with filepath.Match against each
	// file's path relative to the job's working directory) for files the
	// worker should keep instead of discarding along with the rest of the
	// working directory. Matched files are archived and uploaded to object
	// storage, and are also visible to any testcase in a stage that
	// (transitively) depends on this testcase's stage, for the lifetime of
	// the submission. Everything else in the working directory - build
	// caches, compiled binaries not explicitly listed, etc. - is deleted
	// once the testcase finishes.
	Artifacts []string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
}

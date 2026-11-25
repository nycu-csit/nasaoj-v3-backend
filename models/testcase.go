package models

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"text/template"

	"github.com/uptrace/bun"
)

type ContentFile struct {
	bun.BaseModel `bun:"table:content_file" json:"-" yaml:"-"`

	SHA1       string `json:"sha1" yaml:"sha1" bun:"sha1,pk"`
	Name       string `json:"name" yaml:"name" bun:"name"`
	Payload    []byte `json:"payload" yaml:"payload,omitempty" bun:"payload"`
	IsTemplate bool   `json:"isTemplate" yaml:"isTemplate,omitempty" bun:"is_template"`

	Template *template.Template `json:"-" yaml:"-" bun:"-"`
}

// contentFileEncoded is ContentFile's wire shape: the (possibly binary)
// payload is carried as a base64 string.
type contentFileEncoded struct {
	SHA1       string `json:"sha1" yaml:"sha1"`
	Name       string `json:"name" yaml:"name"`
	Payload    string `json:"payload" yaml:"payload,omitempty"`
	IsTemplate bool   `json:"isTemplate" yaml:"isTemplate,omitempty"`
}

func (self ContentFile) encoded() contentFileEncoded {
	return contentFileEncoded{
		SHA1:       self.SHA1,
		Name:       self.Name,
		Payload:    base64.StdEncoding.EncodeToString(self.Payload),
		IsTemplate: self.IsTemplate,
	}
}

// MarshalJSON always emits the payload as a base64 string ("" when empty,
// never null), so a dehydrated file hashes the same regardless of whether
// its payload slice is nil or empty.
func (self ContentFile) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.encoded())
}

func (self *ContentFile) UnmarshalJSON(b []byte) error {
	var in contentFileEncoded
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}
	payload, err := base64.StdEncoding.DecodeString(in.Payload)
	if err != nil {
		return err
	}
	*self = ContentFile{
		SHA1:       in.SHA1,
		Name:       in.Name,
		Payload:    payload,
		IsTemplate: in.IsTemplate,
	}
	return nil
}

func (self ContentFile) MarshalYAML() (interface{}, error) {
	return self.encoded(), nil
}

// ContentFileSHA1 is the content-addressed key of a content file: the sha1
// of its payload's base64 encoding.
func ContentFileSHA1(payload []byte) string {
	h := sha1.Sum([]byte(base64.StdEncoding.EncodeToString(payload)))
	return hex.EncodeToString(h[:])
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

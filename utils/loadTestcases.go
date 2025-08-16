package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func LoadTestcases(dir string) (map[string]models.ContentFile, error) {
	ret := make(map[string]models.ContentFile)

	err := filepath.WalkDir(dir,
		func(cur string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			b, err := os.ReadFile(cur)
			if err != nil {
				return fmt.Errorf("file '%s' read failed, %w", cur, err)
			}

			name, err := filepath.Rel(dir, cur)
			if err != nil {
				return fmt.Errorf("parse relative path failed, %w", err)
			}
			fmt.Fprintf(os.Stderr, "[INFO] Found '%s'\n", name)

			h := sha1.Sum(b)
			contentFile := models.ContentFile{
				Name:    name,
				SHA1:    hex.EncodeToString(h[:]),
				Payload: string(b[:]),
			}
			if path.Ext(name) == ".tmpl" {
				tmpl, err := template.New(name).Parse(string(b[:]))
				if err != nil {
					return fmt.Errorf("templating failed, %w", err)
				}
				contentFile.IsTemplate = true
				contentFile.Template = tmpl
			} else {
				contentFile.IsTemplate = false
				contentFile.Template = nil
			}
			ret[name] = contentFile

			fmt.Fprintf(os.Stderr, "[INFO] Load testcase '%s'\n", name)
			return nil
		})
	if err != nil {
		return ret, fmt.Errorf("filepath walk failed, %w", err)
	}

	return ret, nil
}

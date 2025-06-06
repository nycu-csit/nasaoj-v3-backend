package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func LoadTestcases(dir string) (*models.TestcaseRepository, error) {

	ret := models.NewTestcaseRepository()

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
			if path.Ext(name) == ".tmpl" {
				tmpl, err := template.New(name).Parse(string(b[:]))
				if err != nil {
					return fmt.Errorf("templating failed, %w", err)
				}
				ret.Tmpl[name] = tmpl
			} else {
				ret.Other[name] = b
			}

			fmt.Fprintf(os.Stderr, "Load testcase '%s'\n", name)
			return nil
		})
	if err != nil {
		return ret, fmt.Errorf("filepath walk failed, %w", err)
	}

	return ret, nil
}

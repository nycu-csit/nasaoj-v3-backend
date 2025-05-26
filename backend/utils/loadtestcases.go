package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

type TestcaseRepository = map[string]*template.Template

func LoadTestcases(dir string) (TestcaseRepository, error) {
	testcases := make(TestcaseRepository)

	err := filepath.WalkDir(dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("file '%s' read failed, %w\n", path, err)
			}

			name, err := filepath.Rel(dir, path)
			if err != nil {
				return fmt.Errorf("parse relative path failed, %w\n", err)
			}
			tmpl, err := template.New(name).Parse(string(b[:]))
			testcases[name] = tmpl
			return nil
		})

	if err != nil {
		return testcases, fmt.Errorf("filepath walk failed, %w\n", err)
	}

	return testcases, nil
}

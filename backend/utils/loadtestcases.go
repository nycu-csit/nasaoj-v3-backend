package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func LoadTestcases(dir string) (map[string]*template.Template, error) {
	testcases := make(map[string]*template.Template)

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

			name, found := strings.CutPrefix(path, dir+"/")
			if !found {
				return fmt.Errorf("path parse failed\n")
			}
			tmpl, err := template.New(name).Parse(string(b[:]))
			testcases[name] = tmpl
			return nil
		})

	if err != nil {
		return testcases, fmt.Errorf("filepath walk failed, %w\n", err)
	}
	fmt.Fprintf(os.Stderr, "testcases: %+v\n", testcases)

	return testcases, nil
}

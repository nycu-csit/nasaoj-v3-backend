package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

func LoadTestcases(dir string) (map[string]*template.Template, error) {
	testcases := make(map[string]*template.Template)

	files, err := os.ReadDir(dir)
	if err != nil {
		return testcases, fmt.Errorf("ReadDir failed, %s\n", err)
	}

	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return testcases,
				fmt.Errorf("file '%s' read failed, %s\n", file.Name(), err)
		}

		tmpl, err := template.New(file.Name()).Parse(string(b[:]))
		testcases[file.Name()] = tmpl
	}

	fmt.Fprintf(os.Stderr, "testcases: %+v\n", testcases)

	return testcases, nil
}

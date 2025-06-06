package utils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"path"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func WriteTar(
	tarbuf *bytes.Buffer,
	contents []models.Content,
	testcases *models.TestcaseRepository,
	args map[string]string,
) error {
	tw := tar.NewWriter(tarbuf)
	for _, content := range contents {
		for _, fileName := range content.Files {
			var b []byte
			if path.Ext(fileName) == ".tmpl" {
				testcase, ok := testcases.Tmpl[fileName]
				if !ok {
					return fmt.Errorf("testcase '%s' not exist", fileName)
				}

				buf := bytes.NewBufferString("")
				if err := testcase.Execute(buf, args); err != nil {
					return fmt.Errorf("template execute failed, %w", err)
				}
				b = buf.Bytes()
			} else {
				var ok bool
				b, ok = testcases.Other[fileName]
				if !ok {
					return fmt.Errorf("testcase '%s' not exist", fileName)
				}
			}

			hdr := &tar.Header{
				Name: fileName,
				Mode: 0600,
				Size: int64(len(b)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("tar header write failed, %w", err)
			}
			if _, err := tw.Write(b); err != nil {
				return fmt.Errorf("tar body write failed, %w", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar writer close failed, %w", err)
	}

	return nil
}

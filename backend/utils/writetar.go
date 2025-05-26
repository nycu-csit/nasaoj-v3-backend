package utils

import (
	"archive/tar"
	"bytes"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func WriteTar(
	tarbuf *bytes.Buffer,
	contents []models.Content,
	testcases TestcaseRepository,
	args map[string]string,
) error {
	tw := tar.NewWriter(tarbuf)
	for _, content := range contents {
		for _, fileName := range content.Files {
			testTmpl, ok := testcases[fileName]
			if !ok {
				return fmt.Errorf("testcase %s not exist\n", fileName)
			}

			b := bytes.NewBufferString("")
			if err := testTmpl.Execute(b, args); err != nil {
				return fmt.Errorf("template execute failed, %w\n", err)
			}

			hdr := &tar.Header{
				Name: fileName,
				Mode: 0600,
				Size: int64(b.Len()),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("tar header write failed, %w\n", err)
			}
			if _, err := tw.Write(b.Bytes()); err != nil {
				return fmt.Errorf("tar body write failed, %w\n", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar writer close failed, %w\n", err)
	}

	return nil
}

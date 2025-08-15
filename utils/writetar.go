package utils

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func WriteTar(
	ctx context.Context,
	problems interfaces.ProblemStorage,
	tarbuf *bytes.Buffer,
	contents []models.Content,
	args map[string]string,
) error {
	tw := tar.NewWriter(tarbuf)
	for _, content := range contents {
		for _, file := range content.Files {

			var b []byte
			if file.IsTemplate {
				buf := bytes.NewBufferString("")
				if err := file.Template.Execute(buf, args); err != nil {
					return fmt.Errorf("template execute failed, %w", err)
				}
				b = buf.Bytes()
				// fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", string(b[:]))
			} else {
				b = []byte(file.Payload)
			}

			hdr := &tar.Header{
				Name: file.Name,
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

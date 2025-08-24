package utils

import (
	"archive/tar"
	"bytes"
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
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
					tracing.Logger.Ctx(ctx).
						Error("failed to execute template",
							zap.Error(err))
					return err
				}
				b = buf.Bytes()
			} else {
				b = []byte(file.Payload)
			}

			hdr := &tar.Header{
				Name: file.Name,
				Mode: 0600,
				Size: int64(len(b)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to write tar header",
						zap.Error(err))
				return err
			}
			if _, err := tw.Write(b); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to write tar body",
						zap.Error(err))
				return err
			}
		}
	}
	if err := tw.Close(); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to close tar writer",
				zap.Error(err))
		return err
	}

	return nil
}

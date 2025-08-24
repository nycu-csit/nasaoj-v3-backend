package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
	"golang.org/x/term"
)

func ReadStdin(
	ctx context.Context,
	original string,
) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		editor := os.Getenv("EDITOR")
		if len(editor) == 0 {
			editor = "vi"
		}

		f, err := os.CreateTemp("", "judge-tmp.yaml")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		defer os.Remove(f.Name())

		fmt.Fprintf(f, "%s\n", original)
		if _, err := f.Seek(0, 0); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to seek to the origin of the file",
					zap.Error(err))
			return "", err
		}

		fmt.Printf("Run: %s %s\n", editor, f.Name())
		cmd := exec.Command(editor, f.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to open editor",
					zap.Error(err))
			return "", err
		}

		b, err := io.ReadAll(f)
		return string(b[:]), err
	} else {
		b, err := io.ReadAll(os.Stdin)
		return string(b[:]), err
	}
}

package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"golang.org/x/term"
)

func ReadStdin(original string) (string, error) {
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
			return "", fmt.Errorf("seek to the origin of the file failed, %w", err)
		}

		fmt.Printf("Run: %s %s\n", editor, f.Name())
		cmd := exec.Command(editor, f.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("editor failed, %w", err)
		}

		b, err := io.ReadAll(f)
		return string(b[:]), err
	} else {
		b, err := io.ReadAll(os.Stdin)
		return string(b[:]), err
	}
}

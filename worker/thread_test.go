package main

import (
	"bytes"
	"testing"
)

func TestRunScript(t *testing.T) {
	t.Run("Run user script should get output", func(t *testing.T) {
		ctx := t.Context()
		script := `echo "Hello, World!"`
		output, err := runScript(ctx, script, 30)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expectedOutput := []byte("Hello, World!\n")
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("expected %q, got %q", expectedOutput, output)
		}
	})
}

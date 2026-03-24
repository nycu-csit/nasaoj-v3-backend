package helper

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [script]",
	Short: "Execute a script in the container working directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptName := args[0]
		baseName := strings.TrimSuffix(scriptName, filepath.Ext(scriptName))

		stdoutPath := filepath.Join("/work", baseName+".stdout")
		stderrPath := filepath.Join("/work", baseName+".stderr")

		stdoutFile, err := os.Create(stdoutPath)
		if err != nil {
			os.Stderr.WriteString("failed to create stdout file: " + err.Error() + "\n")
			os.Exit(1)
		}
		defer stdoutFile.Close()

		stderrFile, err := os.Create(stderrPath)
		if err != nil {
			os.Stderr.WriteString("failed to create stderr file: " + err.Error() + "\n")
			os.Exit(1)
		}
		defer stderrFile.Close()

		bashCmd := exec.Command("bash", scriptName)
		bashCmd.Dir = "/work"
		bashCmd.Stdin = os.Stdin
		bashCmd.Stdout = io.MultiWriter(os.Stdout, stdoutFile)
		bashCmd.Stderr = io.MultiWriter(os.Stderr, stderrFile)

		if err := bashCmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}
	},
}

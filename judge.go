package main

import (
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/backend"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/worker"
)

var rootCmd = &cobra.Command{
	Use:   "judge",
	Short: "NASAOJ",
}

func main() {
	rootCmd.AddCommand(cmd.RunProblemCmd)
	rootCmd.AddCommand(backend.BackendCmd)
	rootCmd.AddCommand(worker.WorkerCmd)
	rootCmd.Execute()
}

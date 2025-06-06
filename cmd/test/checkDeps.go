package test

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var checkDepsCmd = &cobra.Command{
	Use:   "check-deps",
	Short: "Check problem dependencies",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Problems dir: %s\n", viper.GetString("problems-dir"))
		_, err := utils.LoadProblems(viper.GetString("problems-dir"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "load problems failed, %+v", err)
			os.Exit(1)
		}
	},
}

func init() {
	checkDepsCmd.Flags().
		String("problems-dir", "./problems", "Directory to read problems")
}

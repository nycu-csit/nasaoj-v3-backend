package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/utils"
)

var (
	cfgFile string
)

var RootCmd = &cobra.Command{
	Use:   "judge",
	Short: "NASAOJ",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		viper.BindPFlags(cmd.PersistentFlags())
		viper.BindPFlags(cmd.Flags())
	},
}

func init() {
	RootCmd.AddCommand(backendCmd)
	RootCmd.AddCommand(workerCmd)
	RootCmd.AddCommand(utils.RootCmd)
}

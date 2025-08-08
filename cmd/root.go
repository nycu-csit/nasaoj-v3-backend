package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/migrates"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/serve"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/tests"
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
	cobra.EnableTraverseRunHooks = true

	RootCmd.AddCommand(serve.RootCmd)
	RootCmd.AddCommand(tests.RootCmd)
	RootCmd.AddCommand(migrates.RootCmd)
	RootCmd.AddCommand(admin.RootCmd)
}

package tests

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

var RootCmd = &cobra.Command{
	Use:   "test",
	Short: "Test problems and testcases",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("$HOME/.config/judge")
		viper.AddConfigPath("/etc/judge")
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				tracing.Logger.Ctx(ctx).
					Panic("failed to read config file",
						zap.Error(err))
				panic(err)
			}
		}

		tracing.InitTracer("nasaoj-test")
	},
}

func init() {
	RootCmd.AddCommand(depsCmd)

	RootCmd.PersistentFlags().
		String("problems-dir", "./problems", "Directory to read problems")
	RootCmd.PersistentFlags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	RootCmd.PersistentFlags().
		String("tools-dir", "./tools", "Directory to read tools")
	RootCmd.PersistentFlags().
		Bool("zap-production", true, "Use production logger")
	RootCmd.PersistentFlags().
		String("uptrace-dsn", "", "Uptrace DSN (Disabled by default)")
}

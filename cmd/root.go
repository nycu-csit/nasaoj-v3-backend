package cmd

import (
	"fmt"
	"os"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(
		&cfgFile, "config", "", "config file, default /etc/nasaoj/nasaoj.yaml")
	RootCmd.AddCommand(backendCmd)
	RootCmd.AddCommand(workerCmd)
	RootCmd.AddCommand(utils.RootCmd)
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot read config, %s\n", err)
		}
	}
}

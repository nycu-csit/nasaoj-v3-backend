package utils

import (
	"context"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func ReadConfig(ctx context.Context) {
	if viper.GetBool("no-read-config") {
		return
	}
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
}

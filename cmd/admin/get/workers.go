package get

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type workersCmdOutput struct {
	IP   string   `yaml:"ip"`
	Tags []string `yaml:"tags"`

	LastAccess string `yaml:"lastAccess"`
}

var workersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Get workers",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		workerURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/worker/states")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		var res []apiHandlers.GetWorkerStatesOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			Get(workerURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get worker states",
					zap.String("url", workerURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}

		var ret []workersCmdOutput
		for _, output := range res {
			ret = append(ret, workersCmdOutput{
				IP:   output.IP,
				Tags: output.Tags,
				LastAccess: time.Now().Sub(
					time.Unix(output.LastAccess, 0)).String() + " ago",
			})
		}

		b, err := yaml.Marshal(ret)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal workers output",
					zap.Error(err))
			panic(err)
		}
		fmt.Printf("%s", string(b[:]))
	},
}

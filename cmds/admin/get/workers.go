package get

import (
	"io"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type workersCmdOutput struct {
	Name string   `yaml:"name"`
	Ip   string   `yaml:"ip"`
	Tags []string `yaml:"tags"`
	Jobs []string `yaml:"jobs"`

	LastAccess string `yaml:"lastAccess"`
}

var workersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Get workers",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res []api.GetWorkerStatesOutput
		utils.RestyBuilder(ctx, "/api/admin/worker/states",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&res).
					ForceContentType("application/json")
			})

		var ret []workersCmdOutput
		for _, output := range res {
			ret = append(ret, workersCmdOutput{
				Name: output.Name,
				Jobs: output.Jobs,
				Ip:   output.Ip,
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
		io.WriteString(os.Stdout, string(b[:]))
	},
}

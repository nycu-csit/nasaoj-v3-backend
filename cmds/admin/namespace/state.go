package namespace

import (
	"io"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

func setStateHandler(
	isEnabled bool,
) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("invalid namespace id",
					zap.Error(err))
			panic(err)
		}

		resp := utils.RestyBuilder(ctx, "/api/admin/namespace/state",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(api.PutNamespaceStateInput{
						Id:        id,
						IsEnabled: isEnabled,
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	}
}

var enableCmd = &cobra.Command{
	Use:   "enable [namespace_id]",
	Short: "Enable a namespace",
	Args:  cobra.ExactArgs(1),
	Run:   setStateHandler(true),
}

var disableCmd = &cobra.Command{
	Use:   "disable [namespace_id]",
	Short: "Disable a namespace",
	Args:  cobra.ExactArgs(1),
	Run:   setStateHandler(false),
}

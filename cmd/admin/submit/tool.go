package submit

import (
	"io"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var toolCmd = &cobra.Command{
	Use:   "tool [tool_sha1]",
	Short: "Submit tool",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		inputArgs := make(map[string]string)
		for _, arg := range viper.GetStringSlice("arg") {
			strs := strings.Split(arg, "=")
			if len(strs) != 2 {
				tracing.Logger.Ctx(ctx).
					Panic("argument format incorrect",
						zap.String("arg", arg))
				panic(types.WrongFormatError)
			}
			inputArgs[strs[0]] = strs[1]
		}

		var output api.SubmitOutput
		reqFunc := func(r *resty.Request) *resty.Request {
			r.Method = resty.MethodPost
			return r.
				SetBody(api.AdminSubmitInput{
					AsCsId: viper.GetString("csid"),
					SubmitInput: api.SubmitInput{
						RefSHA1:   args[0],
						Type:      models.SubmissionTypeTool,
						InputArgs: inputArgs,
					},
				}).
				SetResult(&output).
				ForceContentType("application/json")
		}
		utils.RestyBuilder(cmd,
			"/api/admin/submit", reqFunc)

		b, err := yaml.Marshal(output)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal output to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}

func init() {
	toolCmd.Flags().
		StringSliceP("arg", "a", []string{}, "Input Args (key=value)")
}

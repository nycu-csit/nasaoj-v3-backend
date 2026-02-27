package get

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

var submissionsCmd = &cobra.Command{
	Use:     "submissions",
	Aliases: []string{"submission", "sub"},
	Short:   "Get submission history",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		pageNum := viper.GetInt64("page-num")
		ownerId := viper.GetString("owner-id")
		refId := viper.GetInt64("ref-id")
		refType := viper.GetInt("type")

		var output api.SubmissionsOutput
		utils.RestyBuilder(ctx, "/api/admin/submissions",
			func(r *resty.Request) *resty.Request {
				if ownerId != "" {
					r = r.SetQueryParam("ownerId", ownerId)
				}
				if refId != 0 {
					r = r.SetQueryParam("refId", strconv.FormatInt(refId, 10))
				}
				return r.
					SetQueryParam("pageNum", strconv.FormatInt(pageNum, 10)).
					SetQueryParam("type", strconv.FormatInt(int64(refType), 10)).
					SetResult(&output).
					ForceContentType("application/json")
			})

		b, err := yaml.Marshal(output.Submissions)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal output to yaml",
					zap.Error(err))
			panic(err)
		}

		io.WriteString(os.Stdout, string(b[:]))
		totalPage := int64(
			((output.PageSize - 1) + output.TotalRowCnt) / output.PageSize)
		fmt.Printf("page: %d/%d\n", min(pageNum+1, totalPage), totalPage)
	},
}

func init() {
	submissionsCmd.Flags().
		Int64("page-num", 0, "Page number")
	submissionsCmd.Flags().
		String("owner-id", "", "OwnerId")
	submissionsCmd.Flags().
		Int64("ref-id", 0, "Reference Id for problems / tools")
	submissionsCmd.Flags().
		Int("type", 0, "Submission Type ('0' for problems, '1' for tools)")
}

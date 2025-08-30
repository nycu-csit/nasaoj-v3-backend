package get

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func outputCmd(
	target string,
) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		reqFunc := func(r *resty.Request) *resty.Request {
			return r.
				SetQueryParam("subId", args[0]).
				SetQueryParam("stage", args[1]).
				SetQueryParam("testcase", args[2])
		}
		resp := utils.RestyBuilder(cmd,
			"/api/admin/"+target, reqFunc)
		io.WriteString(os.Stdout, string(resp.Body()[:]))
	}
}

var stdoutCmd = &cobra.Command{
	Use:   "stdout [submission_id] [stage] [testcase]",
	Short: "Get stdout of a testcase",
	Args:  cobra.ExactArgs(3),
	Run:   outputCmd("stdout"),
}

var stderrCmd = &cobra.Command{
	Use:   "stderr [submission_id] [stage] [testcase]",
	Short: "Get stderr of a testcase",
	Args:  cobra.ExactArgs(3),
	Run:   outputCmd("stderr"),
}

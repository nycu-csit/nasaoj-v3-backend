package get

import (
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type artifactsURLOutput struct {
	URL string `json:"url"`
}

var artifactsOutputPath string

// artifactsCmd downloads a testcase's kept artifacts tar. The backend only
// hands out a short-lived presigned MinIO URL (GET /api/admin/artifacts/url)
// - the actual bytes are fetched directly from object storage, the same way
// the worker already downloads ref/job tars, rather than proxying through
// the backend.
var artifactsCmd = &cobra.Command{
	Use:   "artifacts [submission_id] [stage] [testcase]",
	Short: "Download a testcase's kept artifacts tar",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res artifactsURLOutput
		utils.RestyBuilder(ctx, "/api/admin/artifacts/url",
			func(r *resty.Request) *resty.Request {
				return r.
					SetQueryParam("subId", args[0]).
					SetQueryParam("stage", args[1]).
					SetQueryParam("testcase", args[2]).
					SetResult(&res)
			})

		resp, err := resty.New().R().Get(res.URL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download artifacts: %v\n", err)
			return
		}
		if !resp.IsSuccess() {
			fmt.Fprintf(os.Stderr,
				"failed to download artifacts: status %d\n", resp.StatusCode())
			return
		}

		outPath := artifactsOutputPath
		if outPath == "" {
			outPath = fmt.Sprintf("artifact-%s-%s-%s.tar", args[0], args[1], args[2])
		}
		if err := os.WriteFile(outPath, resp.Body(), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", outPath, err)
			return
		}
		fmt.Printf("Downloaded to %s\n", outPath)
	},
}

func init() {
	artifactsCmd.Flags().StringVarP(&artifactsOutputPath, "output", "o", "",
		"Output file path (default: artifact-<subId>-<stage>-<testcase>.tar)")
}

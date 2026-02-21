package get

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Get login info",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/me", nil)
		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}

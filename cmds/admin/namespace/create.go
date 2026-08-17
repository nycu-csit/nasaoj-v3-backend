package namespace

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var (
	createName  string
	createTitle string
	createArgs  map[string]string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a namespace",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res models.Namespace
		utils.RestyBuilder(ctx, "/api/admin/namespace",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPost
				return r.
					SetResult(&res).
					ForceContentType("application/json").
					SetBody(api.PostNamespaceInput{
						Name:  createName,
						Title: createTitle,
						Args:  createArgs,
					})
			})

		fmt.Printf("Created namespace: id=%d name=%s\n", res.Id, res.Name)
	},
}

func init() {
	createCmd.Flags().
		StringVar(&createName, "name", "", "Namespace name (unique slug)")
	createCmd.Flags().
		StringVar(&createTitle, "title", "", "Namespace human-readable title")
	createCmd.Flags().
		StringToStringVar(&createArgs, "args", nil,
			"Default job args for every submission in this namespace "+
				"(key=value, repeatable) - the outermost layer of the same "+
				"overlay problem/stage/testcase args use, so they can still "+
				"override a namespace default for the same key")
	_ = createCmd.MarkFlagRequired("name")
}

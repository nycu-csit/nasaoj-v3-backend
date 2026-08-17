package gen

import (
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func tokenCmdHandler(
	tokenType string,
	userId int64,
) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		description, err := cmd.Flags().GetString("description")
		if err != nil {
			fmt.Printf("failed to read description: %v\n", err)
			return
		}

		ctx := cmd.Context()
		var res api.PostTokenOutput
		utils.RestyBuilder(ctx, "/api/admin/token",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPost
				return r.
					SetResult(&res).
					ForceContentType("application/json").
					SetBody(api.PostTokenInput{
						TokenType:   tokenType,
						Description: description,
						UserId:      userId,
					})
			})

		fmt.Printf("Token: %s\n", res.Token)
	}
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate Access Token",
}

// addDescriptionFlag registers a required --description flag: it's how
// `judge admin get tokens` tells tokens apart later.
func addDescriptionFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String("description", "",
		"What this token is for (required)")
	cmd.MarkFlagRequired("description")
	return cmd
}

func init() {
	tokenCmd.AddCommand(addDescriptionFlag(&cobra.Command{
		Use:   "admin",
		Short: "Generate Admin Access Token",
		Run:   tokenCmdHandler("admin", 0),
	}))
	tokenCmd.AddCommand(addDescriptionFlag(&cobra.Command{
		Use:   "worker",
		Short: "Generate Worker Access Token",
		Run:   tokenCmdHandler("worker", 0),
	}))
	tokenCmd.AddCommand(addDescriptionFlag(&cobra.Command{
		Use:   "user <userId>",
		Short: "Generate a token scoped to a user's live privileges/namespaces",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			userId, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				fmt.Printf("invalid userId: %v\n", err)
				return
			}
			tokenCmdHandler("user", userId)(cmd, args)
		},
	}))
}

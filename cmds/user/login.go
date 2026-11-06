package user

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login via OIDC, or username/password with --username",
	// No PersistentPreRun of its own: with cobra.EnableTraverseRunHooks the
	// user root's hook already runs for this command, and it skips the
	// stored session when the command is login (see RootCmd).
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Verify login by querying /api/me
		var meOutput api.MeOutput
		utils.RestyBuilder(ctx, "/api/me",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&meOutput).
					ForceContentType("application/json")
			})

		fmt.Printf("Login successful!\n")
		fmt.Printf("Username: %s\n", meOutput.AccessTokenClaim.Resource.Username)
		fmt.Printf("Session saved to: %s\n", GetSessionPath())
	},
}

func init() {
	RootCmd.AddCommand(loginCmd)
}

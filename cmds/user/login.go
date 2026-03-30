package user

import (
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login via OIDC",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Clear existing session to force OIDC login
		path := GetSessionPath()
		if path != "" {
			os.Remove(path)
		}

		// Call parent's PersistentPreRun to handle OIDC
		RootCmd.PersistentPreRun(cmd, args)
	},
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

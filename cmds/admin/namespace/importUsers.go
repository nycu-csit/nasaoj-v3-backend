package namespace

import (
	"encoding/json"
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

// importUsers reads users as YAML from stdin into inputs and PUTs them to
// the namespace import endpoint, printing what was created / added.
func importUsers(cmd *cobra.Command, path, example string, inputs any) {
	ctx := cmd.Context()

	stdin, err := utils.ReadStdin(ctx, example)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal([]byte(stdin), inputs); err != nil {
		tracing.Logger.Ctx(ctx).
			Panic("failed to unmarshal yaml",
				zap.Error(err))
		panic(err)
	}
	b, err := json.Marshal(inputs)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Panic("failed to marshal to json",
				zap.Error(err))
		panic(err)
	}

	var imported []models.NamespaceUserImported
	utils.RestyBuilder(ctx, path,
		func(r *resty.Request) *resty.Request {
			r.Method = resty.MethodPut
			return r.
				SetBody(string(b[:])).
				SetResult(&imported).
				ForceContentType("application/json")
		})

	out, err := yaml.Marshal(imported)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Panic("failed to marshal output to yaml",
				zap.Error(err))
		panic(err)
	}
	io.WriteString(os.Stdout, string(out[:]))
}

var importUsersCmd = &cobra.Command{
	Use:   "import-users",
	Short: "Create users and add them to the namespace (--namespace)",
	Long: `Create users and add them to the namespace (--namespace).

Users whose username already exists are only added to the namespace; their
account (password, args, global admin flag) is left untouched. isAdmin
grants namespace admin of this namespace only.`,
}

var importUsersPlainCmd = &cobra.Command{
	Use:   "plain",
	Short: "Import plain-login users from YAML on stdin",
	Run: func(cmd *cobra.Command, args []string) {
		var inputs []api.PutNamespaceUsersPlainInputUser
		importUsers(cmd, "/api/admin/namespace/users/plain", `---
# Example
# - username: 'phkoan'
#   password: 'verysecurepassword'
#   isAdmin: false
#   args:
#     poop: abc
`, &inputs)
	},
}

var importUsersCSSSOCmd = &cobra.Command{
	Use:   "cssso",
	Short: "Import CSSSO users from YAML on stdin",
	Run: func(cmd *cobra.Command, args []string) {
		var inputs []api.PutNamespaceUsersCSSSOInputUser
		importUsers(cmd, "/api/admin/namespace/users/cssso", `---
# Example
# - csId: '111550001'
#   isAdmin: false
#   args:
#     poop: abc
`, &inputs)
	},
}

func init() {
	importUsersCmd.AddCommand(importUsersPlainCmd)
	importUsersCmd.AddCommand(importUsersCSSSOCmd)
}

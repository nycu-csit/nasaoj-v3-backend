package namespace

import (
	"io"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

var addUsersAdmin bool

func parseUserIds(
	cmd *cobra.Command,
	args []string,
) []int64 {
	ids := make([]int64, len(args))
	for i, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			tracing.Logger.Ctx(cmd.Context()).
				Panic("invalid user id",
					zap.String("arg", arg),
					zap.Error(err))
			panic(err)
		}
		ids[i] = id
	}
	return ids
}

// The target namespace comes from the global --namespace/-n flag (X-Namespace).

var addUsersCmd = &cobra.Command{
	Use:   "add-users [user_id...]",
	Short: "Add users to the namespace (--namespace)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/admin/namespace/users",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(api.PutNamespaceUsersInput{
						UserIds: parseUserIds(cmd, args),
						IsAdmin: addUsersAdmin,
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}

var removeUsersCmd = &cobra.Command{
	Use:   "remove-users [user_id...]",
	Short: "Remove users from the namespace (--namespace)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/admin/namespace/users",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodDelete
				return r.
					SetBody(api.DeleteNamespaceUsersInput{
						UserIds: parseUserIds(cmd, args),
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List members of the namespace (--namespace)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var members []models.NamespaceMember
		utils.RestyBuilder(ctx, "/api/admin/namespace/users",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&members).
					ForceContentType("application/json")
			})

		b, err := yaml.Marshal(members)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal members to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}

func init() {
	addUsersCmd.Flags().
		BoolVar(&addUsersAdmin, "admin", false,
			"Grant namespace-admin role to the added users")
}

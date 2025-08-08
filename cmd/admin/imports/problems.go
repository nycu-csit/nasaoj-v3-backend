package imports

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var problemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Import problems",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			return fmt.Errorf("sqlite init failed, %w", err)
		}
		bunDB := bun.NewDB(sqldb, sqlitedialect.New())
		if err := utils.InitDB(ctx, bunDB); err != nil {
			return fmt.Errorf("db init failed, %w", err)
		}
		db := implements.NewBunDatabase(bunDB)
		problems := implements.NewDBProblemStorage(db)

		err = utils.LoadProblems(ctx, problems,
			viper.GetString("problems-dir"), viper.GetString("testcases-dir"))
		if err != nil {
			panic(err)
		}

		exportProblems, err := problems.GetAllProblems(ctx)
		if err != nil {
			return fmt.Errorf("get all problems failed, %w", err)
		}

		// fmt.Fprintf(os.Stderr, "problems: %+v\n", exportProblems)
		client, ok := ctx.Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/problems")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}
		resp, err := client.R().
			SetBody(exportProblems).
			Put(url)
		if err != nil {
			return fmt.Errorf("PUT %s failed, %w", url, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
		return nil
	},
}

func init() {
	problemsCmd.Flags().
		String("problems-dir", "./problems", "Directory to read problems")
	problemsCmd.Flags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
}

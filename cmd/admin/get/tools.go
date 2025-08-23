package get

import (
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gopkg.in/yaml.v2"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Get tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/tools")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		var res []apiHandlers.ProblemOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			Get(url)
		if err != nil {
			return fmt.Errorf("GET %s failed, %w", url, err)
		}

		// fmt.Printf("%+v\n", resp.Request)

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}

		var ret []problemResult
		for _, problem := range res {
			ret = append(ret, problemResult{
				ProblemId:   problem.ProblemId,
				SHA1:        problem.SHA1,
				ImportedAt:  time.Unix(problem.ImportedAt, 0).String(),
				Name:        problem.Name,
				Description: problem.Description,
				Tag:         problem.Tag,
				IsEnabled:   problem.IsEnabled,
			})
		}

		sort.Slice(ret, func(i, j int) bool {
			return res[i].ProblemId < res[j].ProblemId ||
				(res[i].ProblemId == res[j].ProblemId && res[i].SHA1 < res[j].SHA1)
		})

		b, err := yaml.Marshal(ret)
		if err != nil {
			return fmt.Errorf("yaml marshal failed, %w", err)
		}
		fmt.Printf("%s", string(b[:]))
		return nil
	},
}

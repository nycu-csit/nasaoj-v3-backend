package get

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

var scoresCmd = &cobra.Command{
	Use:     "scores",
	Short:   "Get scores [problem id]",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"score"},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res []models.UserScore
		utils.RestyBuilder(ctx, "/api/admin/dump-scores/"+args[0],
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&res).
					ForceContentType("application/json")
			})

		w := csv.NewWriter(os.Stdout)
		if viper.GetString("file") != "" {
			file, err := os.OpenFile(viper.GetString("file"),
				os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Panic("failed to open file",
						zap.String("file.name", viper.GetString("file")),
						zap.Error(err))
				panic(err)
			}
			w = csv.NewWriter(file)
		}
		defer w.Flush()

		if viper.GetBool("with-time") {
			w.Write([]string{"cs_id", "score", "created_at"})
		} else {
			w.Write([]string{"cs_id", "score"})
		}
		for _, score := range res {
			if viper.GetBool("with-time") {
				w.Write([]string{
					score.CsId,
					strconv.FormatInt(score.Score, 10),
					time.Unix(score.CreatedAt, 0).String(),
				})
			} else {
				w.Write([]string{
					score.CsId,
					strconv.FormatInt(score.Score, 10),
				})
			}
		}
	},
}

func init() {
	scoresCmd.Flags().
		StringP("file", "f", "", "File to save to")
	scoresCmd.Flags().
		Bool("with-time", false, "output with submission time")
}

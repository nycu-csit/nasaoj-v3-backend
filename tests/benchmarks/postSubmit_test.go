package benchmarks

import (
	"context"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func BenchmarkPostSubmitParallel(b *testing.B) {
	ctx := b.Context()
	tracing.InitTracer("nasaoj-test")
	utils.ReadConfig(ctx)

	client := resty.New()
	client.SetAuthToken(viper.GetString("access-token"))
	ctx = context.WithValue(ctx, types.RestyClient(""), client)

	// Fetch problem
	var problems []api.ProblemOutput
	utils.RestyBuilder(ctx, "/api/problems",
		func(r *resty.Request) *resty.Request {
			return r.
				SetResult(&problems).
				ForceContentType("application/json")
		})
	if len(problems) < 1 {
		tracing.Logger.Ctx(ctx).
			Panic("no problems")
		panic(types.ProblemNotExistError)
	}
	problem := problems[0]

	// Import user
	utils.RestyBuilder(ctx, "/api/admin/users",
		func(r *resty.Request) *resty.Request {
			r.Method = resty.MethodPut
			return r.
				SetBody(`[{"csId": "test"}]`)
		})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			utils.RestyBuilder(ctx, "/api/admin/submit",
				func(r *resty.Request) *resty.Request {
					r.Method = resty.MethodPost
					return r.
						SetBody(api.AdminSubmitInput{
							AsCsId: "test",
							SubmitInput: api.SubmitInput{
								RefSHA1: problem.SHA1,
								Type:    models.SubmissionTypeProblem,
							},
						})
				})
		}
	})
}

package benchmarks

import (
	"context"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func BenchmarkGetProblemsParallel(b *testing.B) {
	ctx := b.Context()
	tracing.InitTracer("nasaoj-test")
	utils.ReadConfig(ctx)

	client := resty.New()
	client.SetAuthToken(viper.GetString("access-token"))
	ctx = context.WithValue(ctx, types.RestyClient(""), client)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var res []api.ProblemOutput
			utils.RestyBuilder(ctx, "/api/problems",
				func(r *resty.Request) *resty.Request {
					return r.
						SetResult(&res).
						ForceContentType("application/json")
				})
		}
	})
}

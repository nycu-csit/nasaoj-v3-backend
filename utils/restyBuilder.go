package utils

import (
	"context"
	"net/http"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

func RestyBuilder(
	ctx context.Context,
	path string,
	reqFunc func(*resty.Request) *resty.Request,
) *resty.Response {
	client, ok := ctx.Value(types.RestyClient("")).(*resty.Client)
	if !ok {
		tracing.Logger.Ctx(ctx).
			Panic("resty client not exist")
		panic(types.ContextNotExistError)
	}

	resourceURL, err := url.JoinPath(
		viper.GetString("endpoint"), path)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Panic("failed to construct url",
				zap.Error(err))
		panic(err)
	}

	req := client.R()
	req.Method = resty.MethodGet
	req.URL = resourceURL
	//ForceContentType("application/json")
	if reqFunc == nil {
		reqFunc = func(r *resty.Request) *resty.Request { return r }
	}
	resp, err := reqFunc(req).Send()
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Panic("failed to sent http request",
				zap.String("url", resourceURL),
				zap.Error(err))
		panic(err)
	}

	if err != nil || resp.StatusCode() != http.StatusOK {
		tracing.Logger.Ctx(ctx).
			Panic("resp.IsSuccess failed")
		panic(types.HTTPRequestFailedError)
	}

	return resp
}

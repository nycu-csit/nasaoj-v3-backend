package serve

import (
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
	workerapi "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/workflows"
	"go.uber.org/zap"

	_ "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/docs"
)

//	@title		NASAOJ v3 Worker
//	@version	3.0

// @license.name	0BSD
// @BasePath		/
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "NASAOJ worker server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Initialize Tracing
		appname := "nasaoj-worker"
		if viper.GetString("uptrace-dsn") != "" {
			tracing.InitUptrace(appname, appname)
			defer tracing.DeferUptrace(ctx)
		}
		tracing.InitTracer(appname)
		defer tracing.DeferLogger()

		if err := os.MkdirAll(viper.GetString("running-dir"), 0700); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to mkdir running dir",
					zap.Error(err))
			panic(err)
		}

		// Initialize CRI client if container-runtime is specified
		var criClient *cri.Client
		if criEndpoint := viper.GetString("container-runtime"); criEndpoint != "" {
			var err error
			criClient, err = cri.NewClient(criEndpoint)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Panic("failed to create CRI client",
						zap.String("endpoint", criEndpoint),
						zap.Error(err))
				panic(err)
			}
			defer criClient.Close()
			tracing.Logger.Ctx(ctx).
				Info("CRI client initialized",
					zap.String("endpoint", criEndpoint))
		}

		workerBinary := viper.GetString("worker-binary")
		accessToken := viper.GetString("access-token")

		client := resty.New()
		client.SetHeader("X-Worker-Name", viper.GetString("name"))
		client.SetAuthScheme("Bearer")
		client.SetAuthToken(accessToken)
		client.SetBaseURL(viper.GetString("scheduler-url"))

		subCancel := workflows.NewCancelRegistry()
		active := workflows.NewActiveSubs()

		// Worker API server: jobs call this to put args / scores. It verifies
		// the worker-signed jobToken and forwards to the backend.
		apiHandlers := workerapi.NewHandlers(client)
		apiRouter := bunrouter.New()
		if viper.GetBool("swagger") {
			apiRouter.GET("/swagger/*any",
				bunrouter.HTTPHandlerFunc(
					httpSwagger.Handler(httpSwagger.InstanceName("worker"))))
		}
		jobGroup := apiRouter.NewGroup("").
			Use(middlewares.ErrorHandler).
			Use(middlewares.AccessLog).
			Use(middlewares.CheckJobToken([]byte(accessToken)))
		jobGroup.PUT("/args", apiHandlers.PutArgs)
		jobGroup.PUT("/score", apiHandlers.PutScore)

		go func() {
			addr := viper.GetString("api-listen")
			tracing.Logger.Ctx(ctx).
				Info("starting worker api server",
					zap.String("addr", addr))
			if err := http.ListenAndServe(addr, apiRouter); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("worker api server stopped",
						zap.Error(err))
			}
		}()

		go workflows.Heartbeat(client, ctx, subCancel, active)

		runner := &workflows.Runner{
			Root:         ctx,
			Client:       client,
			Sem:          make(chan struct{}, viper.GetInt64("threads")),
			Secret:       []byte(accessToken),
			Endpoint:     viper.GetString("worker-api-endpoint"),
			SubCancel:    subCancel,
			Active:       active,
			CriClient:    criClient,
			WorkerBinary: workerBinary,
		}

		// Submission claim loop. Concurrent submissions are bounded by the
		// max-submissions semaphore; jobs across all submissions share the
		// thread pool.
		subSem := make(chan struct{}, viper.GetInt64("max-submissions"))
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			subSem <- struct{}{}
			claim, ok := workflows.ClaimSubmission(client, ctx)
			if !ok {
				<-subSem
				time.Sleep(time.Duration(
					viper.GetInt64("thread-backoff")) * time.Second)
				continue
			}

			go func(claim workflows.ClaimResponse) {
				defer func() { <-subSem }()
				runner.RunSubmission(claim)
			}(claim)
		}
	},
}

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "undefined"
	}
	workerCmd.Flags().
		String("name", hostname, "Worker name")
	workerCmd.Flags().
		Int64("threads", 20, "Max concurrent jobs across all submissions")
	workerCmd.Flags().
		Int64("max-submissions", 4, "Max concurrent submissions claimed at once")
	workerCmd.Flags().
		String("api-listen", ":8081", "Listen address for the worker API server")
	workerCmd.Flags().
		Bool("swagger", false, "Enable swagger")
	workerCmd.Flags().
		String("worker-api-endpoint", "http://127.0.0.1:8081",
			"Worker API endpoint advertised to jobs (env WORKER_API_ENDPOINT)")
	workerCmd.Flags().
		Int64("thread-backoff", 2, "Thread backoff seconds")
	workerCmd.Flags().
		Int64("heartbeat-sec", 5, "Heartbeat interval in seconds")
	workerCmd.Flags().
		Int("putjob-retry", 3, "How many times to retry if put job failed")
	workerCmd.Flags().
		String("scheduler-url", "http://127.0.0.1:8080", "Scheduler url")
	workerCmd.Flags().
		String("access-token", "", "Scheduler access token")
	workerCmd.Flags().
		String("tags", "default", "Tags for worker to take jobs")
	workerCmd.Flags().
		String("running-dir", "/var/run/nasaoj-worker", "Worker running directory")
	workerCmd.Flags().
		String("uptrace-dsn", "", "Uptrace DSN (Disabled by default)")
	workerCmd.Flags().
		String("log-level", "", "Log level (debug, info, warn, error)")
	workerCmd.Flags().
		String("container-runtime", "", "CRI endpoint (e.g., unix:///run/crio/crio.sock). Empty disables container support")
	workerCmd.Flags().
		String("worker-binary", "/work/judge", "Path to worker binary for container mounting")
	workerCmd.Flags().
		String("host-path-prefix", "", "Host path prefix for CRI mounts (e.g., if running-dir is /var/run/nasaoj-worker inside container but /host/nasaoj-worker on host, set this to /host/nasaoj-worker)")
	workerCmd.Flags().
		String("host-binary-path", "", "Host path to worker binary for CRI mounts (if different from worker-binary)")
}

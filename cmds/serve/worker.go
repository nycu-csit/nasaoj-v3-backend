package serve

import (
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/workflows"
	"go.uber.org/zap"
)

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

		client := resty.New()
		client.SetHeader("X-Worker-Name", viper.GetString("name"))
		client.SetAuthScheme("Bearer")
		client.SetAuthToken(viper.GetString("access-token"))
		client.SetBaseURL(viper.GetString("scheduler-url"))

		jobIn := make(chan models.JobWithFile)
		jobOut := make(chan models.JobWithFile, viper.GetInt64("threads")*2)

		cancelRegistry := workflows.NewCancelRegistry()
		go workflows.Heartbeat(client, ctx, cancelRegistry)

		go func() { // FetchJob threads
			retCh := make(chan error)
			for {
				go workflows.FetchJob(client, ctx, retCh, jobIn)
				ret := <-retCh
				if ret != nil {
					tracing.Logger.Ctx(ctx).
						Warn("fetch thread end with error",
							zap.Error(ret))
					time.Sleep(time.Duration(
						viper.GetInt64("thread-backoff")) * time.Second)
				}
			}
		}()

		go func() { // PutJob threads
			retCh := make(chan error)
			for {
				go workflows.PutJob(client, ctx, retCh, jobOut)
				ret := <-retCh
				if ret != nil {
					tracing.Logger.Ctx(ctx).
						Error("thread 'PutJob' end with error",
							zap.Error(ret))
				}
			}
		}()

		for i := int64(0); i < viper.GetInt64("threads"); i++ {
			go workflows.WorkerThread(ctx, jobIn, jobOut, cancelRegistry, criClient, workerBinary)
		}

		<-ctx.Done()
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
		Int64("threads", 20, "Threads count")
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

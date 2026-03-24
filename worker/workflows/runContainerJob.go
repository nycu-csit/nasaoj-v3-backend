package workflows

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func RunContainerJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
	cancelRegistry *CancelRegistry,
	criClient *cri.Client,
	workerBinary string,
) {
	tracing.Logger.Ctx(pctx).
		Debug("starting container job",
			zap.Int64("submission_id", job.SubId),
			zap.String("stage", job.StageName),
			zap.String("testcase", job.TestName),
			zap.String("image", job.Image),
			zap.Int64("timeout", job.Timeout))

	refpath := path.Join(viper.GetString("running-dir"), job.RefFilename())
	basepath := path.Join(viper.GetString("running-dir"), job.Id())
	tarbuf, err := os.ReadFile(refpath)
	if err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to open ref tar",
				zap.Error(err))
		retCh <- err
		return
	}

	if err := utils.Rmrf(pctx, basepath); err != nil {
		retCh <- err
		return
	}

	if err := utils.UnTar(pctx, basepath, tarbuf); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to untar ref file",
				zap.Error(err))
		retCh <- err
		return
	}

	if err := utils.UnTar(pctx, basepath, job.Tarbuf); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to untar job file",
				zap.Error(err))
		retCh <- err
		return
	}

	tracing.Logger.Ctx(pctx).
		Debug("extracted job files",
			zap.String("basepath", basepath),
			zap.String("refpath", refpath))

	scriptPath := path.Join(basepath, job.Id()+".bash")
	scriptFile, err := os.Create(scriptPath)
	if err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to open script file",
				zap.Error(err))
		retCh <- err
		return
	}
	if _, err := io.WriteString(scriptFile, job.Script); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to write script into file",
				zap.Error(err))
		retCh <- err
		return
	}
	if err := scriptFile.Close(); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to close script file",
				zap.Error(err))
		retCh <- err
		return
	}

	inputPath := path.Join(basepath, job.Id()+".input")
	inputFile, err := os.Create(inputPath)
	if err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to open input file",
				zap.Error(err))
		retCh <- err
		return
	}
	if _, err := io.WriteString(inputFile, job.Input); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to write input into file",
				zap.Error(err))
		retCh <- err
		return
	}
	if err := inputFile.Close(); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to close input file",
				zap.Error(err))
		retCh <- err
		return
	}

	tracing.Logger.Ctx(pctx).
		Debug("prepared script and input files",
			zap.String("script_path", scriptPath),
			zap.String("input_path", inputPath))

	exists, err := criClient.ImageExists(pctx, job.Image)
	if err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to check image existence",
				zap.String("image", job.Image),
				zap.Error(err))
		retCh <- err
		return
	}

	if !exists {
		tracing.Logger.Ctx(pctx).
			Info("pulling image",
				zap.String("image", job.Image))
		if err := criClient.PullImage(pctx, job.Image); err != nil {
			tracing.Logger.Ctx(pctx).
				Error("failed to pull image",
					zap.String("image", job.Image),
					zap.Error(err))
			retCh <- err
			return
		}
	}

	job.ExecAt = time.Now().Unix()
	duration := time.Duration(job.Timeout) * time.Second

	ctx, cancel := context.WithTimeout(pctx, duration)
	defer cancel()

	if cancelRegistry != nil {
		cancelRegistry.Register(job.Id(), cancel)
		defer cancelRegistry.Unregister(job.Id())
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", job.SubId),
		attribute.String("stage.name", job.StageName),
		attribute.String("testcase.name", job.TestName),
		attribute.String("container.image", job.Image))

	// Calculate host paths for CRI mounts
	// When worker runs in Docker but CRI-O runs on host, paths need translation
	hostBasepath := basepath
	if hostPrefix := viper.GetString("host-path-prefix"); hostPrefix != "" {
		runningDir := viper.GetString("running-dir")
		hostBasepath = path.Join(hostPrefix, job.Id())
		tracing.Logger.Ctx(pctx).
			Debug("using host path prefix for CRI mount",
				zap.String("running_dir", runningDir),
				zap.String("host_prefix", hostPrefix),
				zap.String("host_basepath", hostBasepath))
	}

	hostBinary := workerBinary
	if hostBinaryPath := viper.GetString("host-binary-path"); hostBinaryPath != "" {
		hostBinary = hostBinaryPath
	}

	// Validate that the binary path is set and exists as a file
	if hostBinary == "" {
		tracing.Logger.Ctx(pctx).
			Error("worker-binary or host-binary-path must be set for container jobs")
		retCh <- fmt.Errorf("worker-binary or host-binary-path must be set for container jobs")
		return
	}

	// Check if the binary exists and is a file (not a directory)
	// This only works when worker runs directly on the host; when worker runs in a container,
	// host-binary-path must be set correctly to point to the binary on the host filesystem
	if viper.GetString("host-binary-path") == "" {
		if fi, err := os.Stat(hostBinary); err != nil {
			tracing.Logger.Ctx(pctx).
				Error("worker binary not found",
					zap.String("path", hostBinary),
					zap.Error(err))
			retCh <- fmt.Errorf("worker binary not found at %s: %w", hostBinary, err)
			return
		} else if fi.IsDir() {
			tracing.Logger.Ctx(pctx).
				Error("worker binary path is a directory, not a file",
					zap.String("path", hostBinary))
			retCh <- fmt.Errorf("worker binary path %s is a directory, not a file", hostBinary)
			return
		}
	}

	containerConfig := &cri.ContainerConfig{
		Name:       job.Id(),
		Image:      job.Image,
		Command:    []string{"/judge", "helper", "exec", job.Id() + ".bash"},
		WorkingDir: "/work",
		Mounts: []cri.Mount{
			{
				HostPath:      hostBasepath,
				ContainerPath: "/work",
				ReadOnly:      false,
			},
			{
				HostPath:      hostBinary,
				ContainerPath: "/judge",
				ReadOnly:      true,
			},
		},
		Labels: map[string]string{
			"nasaoj.job.id":        job.Id(),
			"nasaoj.submission.id": fmt.Sprintf("%d", job.SubId),
			"nasaoj.stage.name":    job.StageName,
			"nasaoj.testcase.name": job.TestName,
		},
	}

	tracing.Logger.Ctx(ctx).
		Debug("creating container",
			zap.String("container_name", containerConfig.Name),
			zap.String("image", containerConfig.Image),
			zap.Strings("command", containerConfig.Command),
			zap.String("working_dir", containerConfig.WorkingDir),
			zap.Duration("timeout", duration))

	resultCh := make(chan *cri.ContainerResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := criClient.CreateAndRunContainer(ctx, containerConfig)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	var ret error = nil
	select {
	case <-ctx.Done():
		job.RetCode = -1
		isTimeout := ctx.Err() == context.DeadlineExceeded
		if isTimeout {
			tracing.Logger.Ctx(ctx).
				Warn("container job timeout")
			job.Status = models.StatusTimeout
		} else {
			tracing.Logger.Ctx(ctx).
				Warn("container job cancelled")
			job.Status = models.StatusCancelled
		}
	case err := <-errCh:
		tracing.Logger.Ctx(ctx).
			Error("container job failed",
				zap.Error(err))
		ret = err
	case result := <-resultCh:
		if result.ExitCode != 0 {
			job.RetCode = int(result.ExitCode)
			job.Status = models.StatusFailed
		} else {
			job.RetCode = 0
			job.Status = models.StatusSuccess
		}
		job.Stdout = result.Stdout
		job.Stderr = result.Stderr
		tracing.Logger.Ctx(ctx).
			Debug("container finished",
				zap.Int("exit_code", int(result.ExitCode)),
				zap.String("status", string(job.Status)),
				zap.Int("stdout_len", len(result.Stdout)),
				zap.Int("stderr_len", len(result.Stderr)))
	}

	stdoutPath := path.Join(basepath, job.Id()+".stdout")
	stderrPath := path.Join(basepath, job.Id()+".stderr")

	if stdoutData, err := os.ReadFile(stdoutPath); err == nil {
		job.Stdout = string(stdoutData)
	}
	if stderrData, err := os.ReadFile(stderrPath); err == nil {
		job.Stderr = string(stderrData)
	}

	tracing.Logger.Ctx(pctx).
		Debug("container job completed",
			zap.String("job_id", job.Id()),
			zap.String("status", string(job.Status)),
			zap.Int("ret_code", job.RetCode),
			zap.Int("stdout_len", len(job.Stdout)),
			zap.Int("stderr_len", len(job.Stderr)))

	retCh <- ret
}

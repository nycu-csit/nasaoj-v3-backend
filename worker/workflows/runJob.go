package workflows

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func RunJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
) {
	// Setup environment
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

	job.ExecAt = time.Now().Unix()
	duration := time.Duration(job.Timeout) * time.Second

	ctx, cancel := context.WithTimeout(pctx, duration)
	defer cancel()

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", job.SubId),
		attribute.String("stage.name", job.StageName),
		attribute.String("testcase.name", job.TestName))

	// Run script
	cmd := exec.CommandContext(ctx, "bash", job.Id()+".bash")
	cmd.Dir = path.Join(basepath)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err = cmd.Start(); err != nil {
		retCh <- err
		return
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var ret error = nil
	select {
	case <-ctx.Done(): // timeout
		job.Status = models.StatusTimeout
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				tracing.Logger.Ctx(ctx).
					Warn("worker thread timeout",
						zap.Error(err))
				ret = err
			}
		}
		tracing.Logger.Ctx(ctx).Warn("worker thread timeout")
	case err := <-done:
		if err != nil {
			switch t := err.(type) {
			case *exec.ExitError:
				job.RetCode = t.ExitCode()
				job.Status = models.StatusFailed
			default:
				tracing.Logger.Ctx(ctx).
					Error("worker thread failed to exit with code",
						zap.Error(err))
				ret = err
			}
		} else {
			job.Status = models.StatusSuccess
		}
	}

	job.Stdout = string(stdoutBuf.Bytes()[:])
	job.Stderr = string(stderrBuf.Bytes()[:])

	retCh <- ret
}

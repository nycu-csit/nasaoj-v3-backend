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
	"go.uber.org/zap"
)

func RunJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
) {
	// Setup environment
	basepath := path.Join(viper.GetString("running-dir"), job.Id())
	if err := utils.UnTar(pctx, basepath, job.Tarbuf); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to untar job",
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

	io.WriteString(scriptFile, job.Script)
	if err := scriptFile.Close(); err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to write script to file",
				zap.Error(err))
		retCh <- err
		return
	}

	job.ExecAt = time.Now().Unix()
	duration := time.Duration(job.SoftTL) * time.Second

	ctx, cancel := context.WithTimeout(pctx, duration)
	defer cancel()

	// Run script
	cmd := exec.CommandContext(ctx, "bash", job.Id()+".bash")
	cmd.Dir = path.Join(basepath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		tracing.Logger.Ctx(pctx).
			Error("failed to get stdin",
				zap.Error(err))
		retCh <- err
		return
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, job.Stdin)
	}()

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
				ret = err
			}
		}
	case err := <-done:
		if err != nil {
			switch t := err.(type) {
			case *exec.ExitError:
				job.RetCode = t.ExitCode()
				job.Status = models.StatusFailed
			default:
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

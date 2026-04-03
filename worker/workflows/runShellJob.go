package workflows

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

func RunShellJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
	cancelRegistry *CancelRegistry,
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

	// Use 2*timeout for the hard deadline (SIGKILL)
	ctx, cancel := context.WithTimeout(pctx, 2*duration)
	defer cancel()

	// Register cancel function so heartbeat can cancel this job
	if cancelRegistry != nil {
		cancelRegistry.Register(job.Id(), cancel)
		defer cancelRegistry.Unregister(job.Id())
	}

	// Run script (don't use CommandContext since we handle signals manually)
	cmd := exec.Command("bash", job.Id()+".bash")
	cmd.Dir = path.Join(basepath)
	// Create a new process group so we can kill all child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err = cmd.Start(); err != nil {
		retCh <- err
		return
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Timer for SIGTERM at 1*timeout
	termTimer := time.NewTimer(duration)
	defer termTimer.Stop()

	var ret error = nil
	termSent := false

	for {
		select {
		case <-termTimer.C: // soft timeout - send SIGTERM to process group
			if cmd.Process != nil && !termSent {
				termSent = true
				tracing.Logger.Ctx(ctx).
					Warn("worker thread soft timeout, sending SIGTERM to process group")
				// Send SIGTERM to the entire process group (negative PID)
				if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
					tracing.Logger.Ctx(ctx).
						Error("worker thread failed to send SIGTERM to process group",
							zap.Error(err))
				}
			}
		case <-ctx.Done(): // hard timeout (2*timeout) or cancel - send SIGKILL to process group
			job.RetCode = -1
			isTimeout := ctx.Err() == context.DeadlineExceeded
			if isTimeout {
				tracing.Logger.Ctx(ctx).
					Warn("worker thread hard timeout, sending SIGKILL to process group")
				job.Status = models.StatusTimeout
			} else {
				tracing.Logger.Ctx(ctx).
					Warn("worker thread cancelled")
				job.Status = models.StatusCancelled
			}

			if cmd.Process != nil {
				// Send SIGKILL to the entire process group (negative PID)
				if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
					tracing.Logger.Ctx(ctx).
						Error("worker thread failed to kill process group",
							zap.Error(err))
					ret = err
				}
			}
			goto done
		case err := <-done:
			if err != nil {
				switch t := err.(type) {
				case *exec.ExitError:
					job.RetCode = t.ExitCode()
					// If SIGTERM was sent, it's a timeout, not a regular failure
					if termSent {
						job.Status = models.StatusTimeout
					} else {
						job.Status = models.StatusFailed
					}
				default:
					tracing.Logger.Ctx(ctx).
						Error("worker thread failed to exit with code",
							zap.Error(err))
					ret = err
				}
			} else {
				job.Status = models.StatusSuccess
			}
			goto done
		}
	}
done:

	job.Stdout = string(stdoutBuf.Bytes()[:])
	job.Stderr = string(stderrBuf.Bytes()[:])

	retCh <- ret
}

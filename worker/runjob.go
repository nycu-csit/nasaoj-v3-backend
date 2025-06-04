package worker

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func RunJob(
	pctx context.Context,
	retCh chan error,
	jobIn chan models.JobWithFile,
	jobOut chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	job := <-jobIn

	// Setup environment
	filepath := path.Join(viper.GetString("running-dir"), job.Id())
	err := os.MkdirAll(filepath, 0700)
	if err != nil && !os.IsExist(err) {
		retCh <- err
		return
	}

	buf := bytes.NewBuffer(job.Tarbuf)
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			retCh <- err
			return
		}

		b, err := io.ReadAll(tr)
		if err != nil {
			retCh <- err
			return
		}

		filepath := path.Join(viper.GetString("running-dir"), job.Id(), hdr.Name)
		err = os.MkdirAll(path.Dir(filepath), 0700)
		if err != nil && !os.IsExist(err) {
			retCh <- err
			return
		}

		err = os.WriteFile(filepath, b, os.FileMode(hdr.Mode))
		if err != nil {
			retCh <- err
			return
		}
	}
	job.ExecAt = time.Now().Unix()
	duration := time.Duration(job.SoftTL) * time.Second
	cmdCtx, cmdCancel := context.WithTimeout(ctx, duration)
	defer cmdCancel()

	// Run script
	cmd := exec.CommandContext(cmdCtx, "bash")
	cmd.Dir = path.Join(viper.GetString("running-dir"), job.Id())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		retCh <- err
		return
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, job.Script)
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
	case <-cmdCtx.Done(): // timeout
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

	jobOut <- job
	retCh <- ret
}

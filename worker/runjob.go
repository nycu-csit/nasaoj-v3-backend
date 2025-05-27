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
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		retCh <- err
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		retCh <- err
		return
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, job.Script)
	}()

	if err = cmd.Start(); err != nil {
		retCh <- err
		return
	}

	stdoutContent, err := io.ReadAll(stdout)
	if err != nil {
		retCh <- err
		return
	}
	job.Stdout = string(stdoutContent[:])

	stderrContent, err := io.ReadAll(stderr)
	if err != nil {
		retCh <- err
		return
	}
	job.Stderr = string(stderrContent[:])

	err = cmd.Wait()
	switch t := err.(type) {
	case *exec.ExitError:
		job.RetCode = t.ExitCode()
	default:
		retCh <- err
	}

	jobOut <- job
	retCh <- nil
}

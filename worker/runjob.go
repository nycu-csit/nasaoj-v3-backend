package main

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/env"
)

func RunJob(
	pctx context.Context,
	ret chan int,
	jobin chan models.JobWithFile,
	jobout chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	job := <-jobin

	// Setup environment
	buf := bytes.NewBuffer(job.Tarbuf)
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ret <- STATUS_TAR_PARSE_FAILED
			return
		}

		b, err := io.ReadAll(tr)
		if err != nil {
			ret <- STATUS_BODY_READ_FAILED
			return
		}

		filepath := path.Join(env.RUNNING_DIR, job.Id(), hdr.Name)
		err = os.MkdirAll(path.Dir(filepath), 0700)
		if err != nil && !os.IsExist(err) {
			ret <- STATUS_DIR_CREATE_FAILED
			return
		}

		err = os.WriteFile(filepath, b, os.FileMode(hdr.Mode))
		if err != nil {
			ret <- STATUS_FILE_WRITE_FAILED
			return
		}
	}

	job.ExecAt = time.Now().Unix()
	duration := time.Now().Add(time.Duration(job.SoftTL) * time.Second)
	cmdCtx, cancel := context.WithDeadline(ctx, duration)
	defer cancel()

	// Run script
	cmd := exec.CommandContext(cmdCtx, "bash")
	cmd.Dir = path.Join(env.RUNNING_DIR, job.Id())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		ret <- STATUS_EXEC_PREP_FAILED
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ret <- STATUS_EXEC_PREP_FAILED
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		ret <- STATUS_EXEC_PREP_FAILED
		return
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, job.Script)
	}()

	if err = cmd.Start(); err != nil {
		ret <- STATUS_EXEC_FAILED
		return
	}

	stdoutContent, err := io.ReadAll(stdout)
	if err != nil {
		ret <- STATUS_CMD_OUTPUT_FAILED
		return
	}
	job.Stdout = string(stdoutContent[:])

	stderrContent, err := io.ReadAll(stderr)
	if err != nil {
		ret <- STATUS_CMD_OUTPUT_FAILED
		return
	}
	job.Stderr = string(stderrContent[:])

	if err = cmd.Wait(); err != nil {
		ret <- STATUS_EXEC_FAILED
		return
	}

	jobout <- job
	ret <- STATUS_OK
}

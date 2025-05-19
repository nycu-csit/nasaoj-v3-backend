package main

import (
	"context"
	"io"
	"os/exec"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func RunJob(
	pctx context.Context,
	ret chan int,
	jobin chan models.Job,
	jobout chan models.Job,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	job := <-jobin

	job.ExecAt = time.Now().Unix()
	duration := time.Now().Add(time.Duration(job.SoftTL) * time.Second)
	cmdCtx, cancel := context.WithDeadline(ctx, duration)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash")
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
		io.WriteString(stdin, job.TestContent)
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

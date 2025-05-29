package workflows

import (
	"context"
	"io"
	"os/exec"
	"path"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func RunJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
) error {
	// Setup environment
	basepath := path.Join(viper.GetString("running-dir"), job.Id())
	if err := utils.UnTar(basepath, job.Tarbuf); err != nil {
		return err
	}

	job.ExecAt = time.Now().Unix()
	duration := time.Duration(job.SoftTL) * time.Second

	ctx, cancel := context.WithTimeout(pctx, duration)
	defer cancel()

	// Run script
	cmd := exec.CommandContext(ctx, "bash")
	cmd.Dir = path.Join(basepath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, job.Script)
	}()

	if err = cmd.Start(); err != nil {
		return err
	}

	stdoutContent, err := io.ReadAll(stdout)
	if err != nil {
		return err
	}
	job.Stdout = string(stdoutContent[:])

	stderrContent, err := io.ReadAll(stderr)
	if err != nil {
		return err
	}
	job.Stderr = string(stderrContent[:])

	err = cmd.Wait()
	switch t := err.(type) {
	case *exec.ExitError:
		job.RetCode = t.ExitCode()
	default:
		return err
	}

	return nil
}

package workflows

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type jobOutType struct {
	StageName string
	TestName  string

	Status models.JobStatus

	RetCode int
	ExecAt  int64
}

func OutputThread(
	_ context.Context,
	jobOut chan models.JobWithFile,
	jobDone chan models.JobWithFile,
) {
	color.Magenta("Output thread is running\n")
	for {
		job, ok := <-jobOut
		if !ok {
			close(jobDone)
			return
		}

		fmt.Printf("\n")
		color.Yellow("job: %+v\n", jobOutType{
			StageName: job.StageName,
			TestName:  job.TestName,

			Status: job.Status,

			RetCode: job.RetCode,
			ExecAt:  job.ExecAt,
		})

		color.Cyan("===== Script =====\n")
		color.White("%s\n", job.Script)
		color.Cyan("===== Stdout =====\n")
		color.White("%s\n", job.Stdout)
		color.Cyan("===== Stderr =====\n")
		color.White("%s\n", job.Stderr)
		color.Cyan("==================\n")

		if job.Status == models.StatusSuccess {
			color.Green("Job Succeed\n")
		} else {
			color.Red("Job Failed\n")
		}

		jobDone <- job
	}
}

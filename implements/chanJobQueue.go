package implements

import (
	"context"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type ChanJobQueue struct {
	data   map[string]chan models.Job
	jobCnt map[string]int
	jobDep map[string]int
}

func NewChanJobQueue() *ChanJobQueue {
	return &ChanJobQueue{
		data:   make(map[string]chan models.Job),
		jobCnt: make(map[string]int),
		jobDep: make(map[string]int),
	}
}

func (self *ChanJobQueue) QueueKey(tag string) string {
	return "queue-" + tag
}

func (self *ChanJobQueue) GetJob(
	ctx context.Context,
	tag string,
) (models.Job, error) {
	if _, exist := self.data[tag]; !exist {
		self.data[tag] = make(chan models.Job, viper.GetInt64("buffer-size"))
	}
	ret, ok := <-self.data[tag]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Warn("channel closed")
		return models.Job{}, types.ChannelClosedError
	}
	return ret, nil
}

func (self *ChanJobQueue) PutJob(
	_ context.Context,
	job models.Job,
	tag string,
) error {
	if _, exist := self.data[tag]; !exist {
		self.data[tag] = make(chan models.Job, viper.GetInt64("buffer-size"))
	}

	self.data[tag] <- job
	return nil
}

func (self *ChanJobQueue) SetJobCnt(
	_ context.Context,
	stageId string,
	cnt int,
) error {
	self.jobCnt[stageId] = cnt
	return nil
}

func (self *ChanJobQueue) SetDepCnt(
	_ context.Context,
	stageId string,
	cnt int,
) error {
	self.jobDep[stageId] = cnt
	return nil
}

func (self *ChanJobQueue) JobDone(
	_ context.Context,
	stageId string,
) (int64, error) {
	self.jobCnt[stageId] -= 1
	return int64(self.jobCnt[stageId]), nil
}

func (self *ChanJobQueue) StageDone(
	_ context.Context,
	stageId string,
) (int64, error) {
	self.jobDep[stageId] -= 1
	return int64(self.jobDep[stageId]), nil
}

// Custom functions
func (self *ChanJobQueue) Size(tag string) int {
	return len(self.data[tag])
}

func (self *ChanJobQueue) Close(tag string) {
	close(self.data[tag])
}

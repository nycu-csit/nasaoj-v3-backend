package implements

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ChanJobQueue struct {
	data   map[models.TagType]chan models.Job
	jobCnt map[string]int
	jobDep map[string]int
}

func (self *ChanJobQueue) Size(tag models.TagType) int {
	return len(self.data[tag])
}

func NewChanJobQueue() *ChanJobQueue {
	return &ChanJobQueue{
		data:   make(map[models.TagType]chan models.Job),
		jobCnt: make(map[string]int),
		jobDep: make(map[string]int),
	}
}

func (self *ChanJobQueue) QueueKey(tag models.TagType) string {
	return fmt.Sprintf("queue-%s", tag)
}

func (self *ChanJobQueue) GetJob(
	_ context.Context,
	tag models.TagType,
) (models.Job, error) {
	if _, exist := self.data[tag]; !exist {
		self.data[tag] = make(chan models.Job, viper.GetInt64("buffer-size"))
	}
	return <-self.data[tag], nil
}

func (self *ChanJobQueue) PutJob(
	_ context.Context,
	job models.Job,
	tag models.TagType,
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

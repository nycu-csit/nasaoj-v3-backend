package implements

import (
	"context"
	"maps"
	"sync"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

// Local is an in-process, single-backend implementation of JobQueue
// (resource-group locks) and WorkerStatusStorage (worker status/lease
// bookkeeping), used in --single-node mode instead of Redis. State does not
// survive a restart and is not shared across processes; that's fine because
// single-node deployments only ever run one backend.
type Local struct {
	mu sync.Mutex

	locks map[string]localLock

	workers    models.WorkerStatusMap
	workerJobs map[string]map[string]struct{}
	cancelled  map[int64]struct{}
}

type localLock struct {
	owner    string
	expireAt time.Time
}

func NewLocal() *Local {
	return &Local{
		locks:      make(map[string]localLock),
		workers:    make(models.WorkerStatusMap),
		workerJobs: make(map[string]map[string]struct{}),
		cancelled:  make(map[int64]struct{}),
	}
}

// -- JobQueue: resource-group locks --

func (self *Local) Lock(
	_ context.Context,
	job *models.Job,
) error {
	if job.ResourceGroup == "" {
		return nil
	}

	self.mu.Lock()
	defer self.mu.Unlock()

	key := resourceGroupKey(job)
	now := time.Now()
	if existing, ok := self.locks[key]; ok && existing.expireAt.After(now) {
		return types.LockAcquisitionError
	}

	self.locks[key] = localLock{
		owner:    job.Id(),
		expireAt: now.Add(time.Duration(job.Timeout+60) * time.Second),
	}
	return nil
}

func (self *Local) Unlock(
	_ context.Context,
	job *models.Job,
) error {
	if job.ResourceGroup == "" {
		return nil
	}

	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.locks, resourceGroupKey(job))
	return nil
}

// -- WorkerStatusStorage --

func (self *Local) GetWorkers(
	_ context.Context,
) (models.WorkerStatusMap, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	ret := make(models.WorkerStatusMap, len(self.workers))
	maps.Copy(ret, self.workers)
	return ret, nil
}

func (self *Local) UpdateWorkerStatus(
	_ context.Context,
	name string,
	ip string,
	tags []string,
) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.workers[name] = models.WorkerStatus{
		Ip:         ip,
		Tags:       tags,
		LastAccess: time.Now().Unix(),
	}
	return nil
}

func (self *Local) UpdateLastAccess(
	_ context.Context,
	name string,
) error {
	if len(name) == 0 {
		return types.WrongFormatError
	}

	self.mu.Lock()
	defer self.mu.Unlock()

	status := self.workers[name]
	status.LastAccess = time.Now().Unix()
	self.workers[name] = status
	return nil
}

func (self *Local) DeleteWorker(
	_ context.Context,
	name string,
) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	delete(self.workers, name)
	delete(self.workerJobs, name)
	return nil
}

func (self *Local) AddWorkerJob(
	_ context.Context,
	workerName string,
	jobId string,
) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	jobs, ok := self.workerJobs[workerName]
	if !ok {
		jobs = make(map[string]struct{})
		self.workerJobs[workerName] = jobs
	}
	jobs[jobId] = struct{}{}
	return nil
}

func (self *Local) RemoveWorkerJob(
	_ context.Context,
	workerName string,
	jobId string,
) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	delete(self.workerJobs[workerName], jobId)
	return nil
}

func (self *Local) GetWorkerJobs(
	_ context.Context,
	workerName string,
) ([]string, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	jobs := self.workerJobs[workerName]
	ret := make([]string, 0, len(jobs))
	for id := range jobs {
		ret = append(ret, id)
	}
	return ret, nil
}

func (self *Local) AddCancelledSubmissions(
	_ context.Context,
	subIds []int64,
) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, id := range subIds {
		self.cancelled[id] = struct{}{}
	}
	return nil
}

func (self *Local) GetCancelledSubmissions(
	_ context.Context,
	subIds []int64,
) ([]int64, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	var ret []int64
	for _, id := range subIds {
		if _, ok := self.cancelled[id]; ok {
			ret = append(ret, id)
		}
	}
	return ret, nil
}

func (self *Local) RemoveCancelledSubmissions(
	_ context.Context,
	subIds []int64,
) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	for _, id := range subIds {
		delete(self.cancelled, id)
	}
	return nil
}

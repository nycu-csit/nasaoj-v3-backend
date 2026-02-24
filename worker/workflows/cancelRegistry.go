package workflows

import (
	"context"
	"sync"
)

type CancelRegistry struct {
	mu      sync.RWMutex
	cancels map[string]context.CancelFunc
}

func NewCancelRegistry() *CancelRegistry {
	return &CancelRegistry{
		cancels: make(map[string]context.CancelFunc),
	}
}

func (r *CancelRegistry) Register(jobId string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancels[jobId] = cancel
}

func (r *CancelRegistry) Unregister(jobId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancels, jobId)
}

func (r *CancelRegistry) Cancel(jobId string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cancel, ok := r.cancels[jobId]; ok {
		cancel()
		delete(r.cancels, jobId)
		return true
	}
	return false
}

func (r *CancelRegistry) CancelAll(jobIds []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, jobId := range jobIds {
		if cancel, ok := r.cancels[jobId]; ok {
			cancel()
			delete(r.cancels, jobId)
		}
	}
}

package workflows

import "sync"

// ActiveSubs is the thread-safe set of submissions this worker currently owns.
// It is reported in heartbeats so the backend can tell the worker which of them
// have been cancelled.
type ActiveSubs struct {
	mu   sync.RWMutex
	subs map[int64]struct{}
}

func NewActiveSubs() *ActiveSubs {
	return &ActiveSubs{subs: make(map[int64]struct{})}
}

func (a *ActiveSubs) Add(subId int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.subs[subId] = struct{}{}
}

func (a *ActiveSubs) Remove(subId int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.subs, subId)
}

func (a *ActiveSubs) List() []int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ret := make([]int64, 0, len(a.subs))
	for id := range a.subs {
		ret = append(ret, id)
	}
	return ret
}

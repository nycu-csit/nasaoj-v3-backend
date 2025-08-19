package models

type WorkerStatus struct {
	Tags       []string `json:"tags"`
	LastAccess int64    `json:"lastAccess"`
}

type WorkerStatusMap = map[string]WorkerStatus

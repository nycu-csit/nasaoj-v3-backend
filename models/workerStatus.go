package models

type WorkerStatus struct {
	Ip         string   `json:"ip"`
	Tags       []string `json:"tags"`
	LastAccess int64    `json:"lastAccess"`
}

type WorkerStatusMap = map[string]WorkerStatus

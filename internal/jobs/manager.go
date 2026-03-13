package jobs

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Job struct {
	ID        string
	Status    Status
	Payload   any
	Error     string
	CreatedAt time.Time
}

type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewManager() *Manager {
	return &Manager{jobs: make(map[string]*Job)}
}

// StartCleanup runs a background goroutine that deletes old jobs.
func (m *Manager) StartCleanup(interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			m.cleanup(maxAge)
		}
	}()
}

func (m *Manager) cleanup(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, job := range m.jobs {
		if time.Since(job.CreatedAt) > maxAge {
			delete(m.jobs, id)
			count++
		}
	}
	if count > 0 {
		slog.Info("Janitor cleaned up old jobs", "count", count)
	}
}

func (m *Manager) CreateJob() string {
	id := uuid.New().String()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[id] = &Job{
		ID:        id,
		Status:    StatusProcessing,
		CreatedAt: time.Now(), // Initialize timestamp
	}
	return id
}

func (m *Manager) GetJob(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	return job, ok
}

func (m *Manager) CompleteJob(id string, data any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok {
		j.Status = StatusCompleted
		j.Payload = data
	}
}

func (m *Manager) FailJob(id string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok {
		j.Status = StatusFailed
		j.Error = errMsg
	}
}

package keywords

import (
	"fmt"
	"path/filepath"

	"keywordAutomator/internal/ai"
	"keywordAutomator/internal/jobs"
	"keywordAutomator/internal/processor"
	"keywordAutomator/internal/source"
)

type KeywordsCommand struct {
	jobMgr    *jobs.Manager
	evaluator ai.KeywordEvaluator
	organizer ai.KeywordOrganizer
}

func (k *KeywordsCommand) Setup(mgr *jobs.Manager, evaluator ai.KeywordEvaluator, organizer ai.KeywordOrganizer) {
	k.jobMgr = mgr
	k.evaluator = evaluator
	k.organizer = organizer
}

// --- Input types ---
// These are serialised from JSON by Wails when the frontend calls the method.

type EvaluationParams struct {
	Filename           string   `json:"filename"`
	CampaignDesc       string   `json:"campaignDesc"`
	ExcludeKeywords    []string `json:"excludeKeywords"`
	CustomInstructions string   `json:"customInstructions"`
}

type ClusterParams struct {
	EvalJobID    string `json:"evalJobId"`
	SelectedRows []int  `json:"selectedRows"` // indices of checked rows
}

// --- Result types ---
// These are serialised to JSON by Wails and received as typed objects in Svelte.

type JobRef struct {
	JobID string `json:"jobId"`
}

type JobStatus struct {
	Status string `json:"status"` // "processing" | "completed" | "failed"
	Error  string `json:"error,omitempty"`
	Data   any    `json:"data,omitempty"`
}

type EvaluationData struct {
	Headers      []string   `json:"headers"`
	Rows         [][]string `json:"rows"`
	OriginalRows [][]string `json:"originalRows"`
}

// --- Methods ---

// StartEvaluation creates a background job and returns its ID immediately.
// The frontend polls GetEvaluationStatus until done.
func (k *KeywordsCommand) StartEvaluation(params EvaluationParams) (JobRef, error) {
	jobID := k.jobMgr.CreateJob()

	go processor.ProcessKeywords(processor.KeywordProcessParams{
		JobID:              jobID,
		Source:             &source.CSVFileSource{Path: filepath.Join("tmp", params.Filename)},
		CampaignDesc:       params.CampaignDesc,
		ExcludeKeywords:    params.ExcludeKeywords,
		CustomInstructions: params.CustomInstructions,
		Evaluator:          k.evaluator,
		Manager:            k.jobMgr,
	})

	return JobRef{JobID: jobID}, nil
}

// GetEvaluationStatus returns the current status of an evaluation job.
func (k *KeywordsCommand) GetEvaluationStatus(jobID string) (JobStatus, error) {
	job, ok := k.jobMgr.GetJob(jobID)
	if !ok {
		return JobStatus{Status: "failed", Error: "job not found"}, nil
	}

	result := JobStatus{Status: string(job.Status)}

	switch job.Status {
	case jobs.StatusCompleted:
		if payload, ok := job.Payload.(processor.KeywordJobPayload); ok {
			result.Data = EvaluationData{
				Headers:      payload.Header,
				Rows:         payload.Content,
				OriginalRows: payload.OriginalRows,
			}
		}
	case jobs.StatusFailed:
		result.Error = job.Error
	}

	return result, nil
}

// StartClustering creates a clustering job from a completed evaluation.
// Takes the eval job's data and the user's selected row indices.
func (k *KeywordsCommand) StartClustering(params ClusterParams) (JobRef, error) {
	job, ok := k.jobMgr.GetJob(params.EvalJobID)
	if !ok || job.Status != jobs.StatusCompleted {
		return JobRef{}, fmt.Errorf("eval job not found or not completed")
	}

	payload, ok := job.Payload.(processor.KeywordJobPayload)
	if !ok {
		return JobRef{}, fmt.Errorf("eval job payload has unexpected type")
	}

	// Filter to selected rows only
	selectedRows := make([][]string, 0, len(params.SelectedRows))
	for _, idx := range params.SelectedRows {
		if idx < len(payload.Content) {
			selectedRows = append(selectedRows, payload.Content[idx])
		}
	}

	jobID := k.jobMgr.CreateJob()

	go processor.ProcessClustering(processor.ClusterProcessParams{
		JobID:        jobID,
		Headers:      payload.Header,
		Rows:         selectedRows,
		OriginalRows: payload.OriginalRows,
		Organizer:    k.organizer,
		Manager:      k.jobMgr,
	})

	return JobRef{JobID: jobID}, nil
}

// GetClusterStatus returns the current status of a clustering job.
func (k *KeywordsCommand) GetClusterStatus(jobID string) (JobStatus, error) {
	job, ok := k.jobMgr.GetJob(jobID)
	if !ok {
		return JobStatus{Status: "failed", Error: "job not found"}, nil
	}

	result := JobStatus{Status: string(job.Status)}

	switch job.Status {
	case jobs.StatusCompleted:
		result.Data = job.Payload
	case jobs.StatusFailed:
		result.Error = job.Error
	}

	return result, nil
}

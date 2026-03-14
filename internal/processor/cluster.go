package processor

import (
	"context"

	"keywordAutomator/internal/ai"
	"keywordAutomator/internal/jobs"
)

type ClusterProcessParams struct {
	Ctx          context.Context
	JobID        string
	Headers      []string
	Rows         [][]string
	OriginalRows [][]string
	Organizer    ai.KeywordOrganizer
	Manager      *jobs.Manager
}

func ProcessClustering(params ClusterProcessParams) {
	newHeaders, newRows, err := params.Organizer.OrganizeKeywords(
		params.Ctx, "", params.Headers, params.Rows,
	)
	if err != nil {
		params.Manager.FailJob(params.JobID, err.Error())
		return
	}

	finalHeaders, finalRows := ZipResults(params.Headers, params.Rows, newHeaders, newRows)

	params.Manager.CompleteJob(params.JobID, ClusterJobPayload{
		Header:       finalHeaders,
		Content:      finalRows,
		OriginalRows: params.OriginalRows,
	})
}

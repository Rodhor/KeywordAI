package processor

import (
	"context"
	"log/slog"

	"keywordAutomator/internal/ai"
	"keywordAutomator/internal/jobs"
	"keywordAutomator/internal/pipeline"
	"keywordAutomator/internal/source"
)

type KeywordProcessParams struct {
	JobID              string
	Source             source.KeywordSource
	ExcludeKeywords    []string
	CampaignDesc       string
	CustomInstructions string
	Evaluator          ai.KeywordEvaluator
	Manager            *jobs.Manager
}

func ProcessKeywords(params KeywordProcessParams) {
	defer func() {
		if err := params.Source.Cleanup(); err != nil {
			slog.Error("Failed to cleanup source", "error", err)
		}
	}()
	slog.Info("Starting work", "id", params.JobID)

	reader, err := params.Source.Open()
	if err != nil {
		slog.Error("Failed to open source", "error", err)
		params.Manager.FailJob(params.JobID, err.Error())
		return
	}
	defer reader.Close()

	keywords, err := ParseCSVToSlice(reader)
	if err != nil {
		slog.Error("Failed to parse CSV", "error", err)
		params.Manager.FailJob(params.JobID, err.Error())
		return
	}

	keywords = DropUnwantedColumns(keywords)
	header, content := ExtractHeaderRow(keywords)

	// Copy original rows to keep track within googlesheets
	originalRows := make([][]string, len(content))
	copy(originalRows, content)

	ctx := &pipeline.Context{Headers: header, Rows: content}

	steps := []pipeline.Step{
		pipeline.NewRemoveEmpty(),
		pipeline.NewDropIfEmpty(0),
		pipeline.NewDeduplicate(0),
		pipeline.NewExcludeRows(params.ExcludeKeywords, 0),
		aiValidationStep{Evaluator: params.Evaluator, Description: params.CampaignDesc, CustomInstructions: params.CustomInstructions},
	}

	if err = pipeline.Run(ctx, steps); err != nil {
		slog.Error("Pipeline failed", "error", err)
		params.Manager.FailJob(params.JobID, err.Error())
		return
	}

	result := KeywordJobPayload{
		Header:       ctx.Headers,
		Content:      ctx.Rows,
		OriginalRows: originalRows,
	}

	params.Manager.CompleteJob(params.JobID, result)
	slog.Info("Job finished successfully", "id", params.JobID)
}

// --- Pipeline Steps ---

type aiValidationStep struct {
	Evaluator          ai.KeywordEvaluator
	Description        string
	CustomInstructions string
}

func (s aiValidationStep) Run(ctx *pipeline.Context) error {
	desc := s.Description
	if s.CustomInstructions != "" {
		desc += "\n\nFor this specific evaluation, also consider: " + s.CustomInstructions
	}
	newHeaders, newRows, err := SendToAIForValidation(s.Evaluator, ctx.Headers, desc, ctx.Rows)
	if err != nil {
		return err
	}
	ctx.Headers = newHeaders
	ctx.Rows = newRows
	return nil
}

// --- Helper Functions ---

func ExtractHeaderRow(keywords [][]string) ([]string, [][]string) {
	if len(keywords) == 0 {
		return nil, nil
	}
	header := keywords[0]
	content := keywords[1:]
	return header, content
}

func DropUnwantedColumns(keywords [][]string) [][]string {
	// Add logic here if you need to slice the rows (e.g., only keep first 5 columns)
	return keywords
}

func SendToAIForValidation(aiClient ai.KeywordEvaluator, headers []string, description string, keywords [][]string) ([]string, [][]string, error) {
	newHeaders, newRows, err := aiClient.EvaluateKeywords(context.Background(), description, headers, keywords)
	if err != nil {
		return nil, nil, err
	}

	// Zip the original headers with the AI's new rows
	finalHeaders, finalRows := ZipResults(headers, keywords, newHeaders, newRows)

	return finalHeaders, finalRows, nil
}

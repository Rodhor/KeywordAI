package ai

import "context"

// KeywordGenerator: Takes a campaign description and returns a new CSV grid of keywords.
type KeywordGenerator interface {
	GenerateKeywords(ctx context.Context, description string) (headers []string, rows [][]string, err error)
}

// KeywordEvaluator: Takes the description and the existing CSV grid,
// returns the grid with an added "Evaluation" column (Good/Bad/Neutral).
type KeywordEvaluator interface {
	EvaluateKeywords(ctx context.Context, description string, headers []string, rows [][]string) (newHeaders []string, newRows [][]string, err error)
}

// KeywordOrganizer: Takes the description and grid,
// returns the grid sorted or with an added "Category" column.
type KeywordOrganizer interface {
	OrganizeKeywords(ctx context.Context, description string, headers []string, rows [][]string) (newHeaders []string, newRows [][]string, err error)
}

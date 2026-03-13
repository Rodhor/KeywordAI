package pipeline

import "strings"

// NewRemoveEmpty removes any row where ALL cells are blank or whitespace-only.
// Use this as a basic sanity pass to drop completely empty rows.
func NewRemoveEmpty() RemoveEmpty {
	return RemoveEmpty{}
}

// RemoveEmpty drops rows where every cell is blank or whitespace-only.
type RemoveEmpty struct{}

func (s RemoveEmpty) Run(ctx *Context) error {
	result := make([][]string, 0, len(ctx.Rows))
	for _, row := range ctx.Rows {
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				result = append(result, row)
				break
			}
		}
	}
	ctx.Rows = result
	return nil
}

// NewDropIfEmpty returns a DropIfEmpty step that drops a row if ANY of the
// specified columns is blank or whitespace-only.
func NewDropIfEmpty(cols ...int) DropIfEmpty {
	return DropIfEmpty{col: cols}
}

// DropIfEmpty drops any row where any of the specified columns is blank or whitespace-only.
type DropIfEmpty struct {
	col []int
}

func (s DropIfEmpty) Run(ctx *Context) error {
	result := make([][]string, 0, len(ctx.Rows))
	for _, row := range ctx.Rows {
		if len(row) == 0 {
			continue
		}
		keep := true
		for _, c := range s.col {
			if c >= len(row) || strings.TrimSpace(row[c]) == "" {
				keep = false
				break
			}
		}
		if keep {
			result = append(result, row)
		}
	}
	ctx.Rows = result
	return nil
}

// NewDeduplicate returns a Deduplicate step that deduplicates on a composite key
// built from the specified columns (case-insensitive).
func NewDeduplicate(cols ...int) Deduplicate {
	return Deduplicate{col: cols}
}

// Deduplicate removes rows with a duplicate key built from the specified columns
// (case-insensitive). The first occurrence is kept.
type Deduplicate struct {
	col []int
}

func (s Deduplicate) Run(ctx *Context) error {
	seen := make(map[string]struct{})
	result := make([][]string, 0, len(ctx.Rows))
	for _, row := range ctx.Rows {
		if len(row) == 0 {
			continue
		}
		parts := make([]string, 0, len(s.col))
		for _, c := range s.col {
			if c < len(row) {
				parts = append(parts, strings.ToLower(strings.TrimSpace(row[c])))
			}
		}
		key := strings.Join(parts, "\x00") // null byte separator avoids false collisions
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, row)
		}
	}
	ctx.Rows = result
	return nil
}

// NewExcludeRows returns an ExcludeRows step that checks the specified columns
// for matches against the given terms.
func NewExcludeRows(terms []string, cols ...int) ExcludeRows {
	return ExcludeRows{Terms: terms, col: cols}
}

// ExcludeRows drops any row where ANY of the specified columns contains one of
// the given terms (case-insensitive substring match).
type ExcludeRows struct {
	Terms []string
	col   []int
}

func (s ExcludeRows) Run(ctx *Context) error {
	result := make([][]string, 0, len(ctx.Rows))
	for _, row := range ctx.Rows {
		if len(row) == 0 {
			continue
		}
		excluded := false
	outer:
		for _, c := range s.col {
			if c >= len(row) {
				continue
			}
			colVal := strings.ToLower(row[c])
			for _, term := range s.Terms {
				if clean := strings.ToLower(strings.TrimSpace(term)); clean != "" && strings.Contains(colVal, clean) {
					excluded = true
					break outer
				}
			}
		}
		if !excluded {
			result = append(result, row)
		}
	}
	ctx.Rows = result
	return nil
}

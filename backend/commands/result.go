package commands

// Result is the standard return type for all command methods.
// The frontend checks Status first, then reads Data or Error.
type Result[T any] struct {
	Status string `json:"status"` // "completed" | "processing" | "failed"
	Data   T      `json:"data"`
	Error  string `json:"error"`
}

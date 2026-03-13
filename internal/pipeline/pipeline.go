package pipeline

// Context is the data flowing through the pipeline between steps.
type Context struct {
	Headers []string
	Rows    [][]string
}

// Step is anything that can transform a Context in place.
type Step interface {
	Run(ctx *Context) error
}

// Run executes each step in order, stopping on the first error.
func Run(ctx *Context, steps []Step) error {
	for _, step := range steps {
		if err := step.Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

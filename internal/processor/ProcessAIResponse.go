package processor

func ZipResults(origHeaders []string, origRows [][]string, aiHeaders []string, aiRows [][]string) ([]string, [][]string) {
	// 1. Merge Headers
	finalHeaders := append(origHeaders, aiHeaders...)

	// 2. Pre-allocate the full results slice
	finalRows := make([][]string, 0, len(origRows))

	// Modern Go 1.22+ range over int
	for i := range len(origRows) {
		// Start with the original row
		row := origRows[i]

		// Check if the AI returned a matching row
		if i < len(aiRows) {
			row = append(row, aiRows[i]...)
		} else {
			// Pad with "N/A" if AI missed the row or hit a limit
			for range len(aiHeaders) {
				row = append(row, "N/A")
			}
		}
		finalRows = append(finalRows, row)
	}

	return finalHeaders, finalRows
}

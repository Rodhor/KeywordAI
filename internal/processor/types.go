package processor

type KeywordJobPayload struct {
	Header       []string
	Content      [][]string
	OriginalRows [][]string
}

type ClusterJobPayload struct {
	Header       []string
	Content      [][]string
	OriginalRows [][]string
}

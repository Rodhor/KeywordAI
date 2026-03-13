package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// AIResponse remains the same, focused on the Delta
type AIResponse struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

type Client struct {
	gvClient       *genai.Client
	model          string
	config         *genai.GenerateContentConfig
	organizerConfig *genai.GenerateContentConfig
}

func NewClient(ctx context.Context) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	var temp float32 = 1.0

	evaluatorSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"headers": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"rows": {
				Type:        genai.TypeArray,
				Description: "One array per input row, containing ONLY 'Evaluation' and 'Reasoning'.",
				Items: &genai.Schema{
					Type:  genai.TypeArray,
					Items: &genai.Schema{Type: genai.TypeString},
				},
			},
		},
		Required: []string{"headers", "rows"},
	}

	organizerSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"headers": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"rows": {
				Type:        genai.TypeArray,
				Description: "One array per input row, containing ONLY the 'Cluster' value.",
				Items: &genai.Schema{
					Type:  genai.TypeArray,
					Items: &genai.Schema{Type: genai.TypeString},
				},
			},
		},
		Required: []string{"headers", "rows"},
	}

	return &Client{
		gvClient: client,
		model:    "gemini-3.1-flash-lite-preview",
		config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: `You are a senior Google Ads specialist with 15+ years of experience managing large-scale paid search campaigns across every major industry. You have expert knowledge of search intent, keyword quality, match types, commercial value, and what separates a high-performing keyword from one that wastes budget.

You will receive a campaign description and a list of keywords that need to be cleaned. The campaign description is the most important input — every keyword must be evaluated against it. A keyword that would perform well in one campaign may be completely wrong for another.

Your job is to evaluate each keyword and decide whether it belongs in this specific campaign or should be removed.

Rules:
- Evaluation must be exactly "Good" or "Bad". No other values are accepted.
- Good: the keyword has intent that aligns with the campaign, is specific enough to attract relevant traffic, and is worth bidding on for this campaign's goals.
- Bad: the keyword is too vague, misaligned with the campaign, navigational to a competitor, clearly non-converting, or likely to attract irrelevant traffic for this campaign.
- Reasoning must be a single short sentence (under 15 words). State the core reason — no padding.
- Match input row count exactly. One output row per input row, in the same order.`}},
			},
			ResponseMIMEType: "application/json",
			ResponseSchema:   evaluatorSchema,
			Temperature:      &temp,
		},
		organizerConfig: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: `You are a Google Ads keyword organizer. Your job is to group keywords by search intent and theme.

Rules:
- Return only a "Cluster" column — one cluster name per row, in the same order as the input.
- Cluster names must be short (2–4 words), title-cased, and descriptive of intent.
- Group by user intent first (e.g. "Buy Running Shoes"), then topic.
- A keyword with no clear cluster gets "Other".
- Match input row count exactly. One output row per input row.`}},
			},
			ResponseMIMEType: "application/json",
			ResponseSchema:   organizerSchema,
			Temperature:      &temp,
		},
	}, nil
}

func (c *Client) EvaluateKeywords(ctx context.Context, desc string, headers []string, rows [][]string) ([]string, [][]string, error) {
	prompt := fmt.Sprintf("Campaign description: %s\n\nEvaluate each keyword for this campaign specifically. Headers: %v\nRows: %v", desc, headers, rows)

	result, err := c.gvClient.Models.GenerateContent(ctx, c.model, genai.Text(prompt), c.config)
	if err != nil {
		return nil, nil, err
	}

	var output AIResponse
	err = json.Unmarshal([]byte(result.Text()), &output)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return output.Headers, output.Rows, nil
}

func (c *Client) OrganizeKeywords(ctx context.Context, desc string, headers []string, rows [][]string) ([]string, [][]string, error) {
	prompt := fmt.Sprintf(
		"Group each keyword into a short thematic cluster based on search intent.\n"+
			"Return only a 'Cluster' header and one cluster name per row, in the same order as input.\n"+
			"Campaign context (optional): %s\n\nHeaders: %v\nRows: %v",
		desc, headers, rows,
	)

	result, err := c.gvClient.Models.GenerateContent(ctx, c.model, genai.Text(prompt), c.organizerConfig)
	if err != nil {
		return nil, nil, err
	}

	var output AIResponse
	err = json.Unmarshal([]byte(result.Text()), &output)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return output.Headers, output.Rows, nil
}

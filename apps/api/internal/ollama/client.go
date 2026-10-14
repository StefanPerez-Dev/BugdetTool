package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL     string
	model       string
	visionModel string
	http        *http.Client
}

func New(baseURL, model, visionModel string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		model:       model,
		visionModel: visionModel,
		http:        &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("ollama status %d", res.StatusCode)
	}
	return nil
}

func (c *Client) Categorize(ctx context.Context, description string, amountCents int, categories []string) (string, error) {
	prompt := fmt.Sprintf(`Je categoriseert Nederlandse banktransacties.
Kies EXACT één categorie uit deze lijst: %s
Transactie: "%s" (bedrag in centen: %d)
Antwoord alleen met de categorienaam, niets anders.`, strings.Join(categories, ", "), description, amountCents)

	body := map[string]any{
		"model":  c.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": 0,
		},
	}
	raw, err := c.post(ctx, "/api/generate", body)
	if err != nil {
		return "", err
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	answer := strings.TrimSpace(out.Response)
	answer = strings.Trim(answer, "\"'` \n\t")
	for _, cat := range categories {
		if strings.EqualFold(cat, answer) {
			return cat, nil
		}
	}
	for _, cat := range categories {
		if strings.Contains(strings.ToLower(answer), strings.ToLower(cat)) {
			return cat, nil
		}
	}
	return "", fmt.Errorf("unrecognized category from ollama: %q", answer)
}

type ExtractedTx struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

func (c *Client) ExtractFromImage(ctx context.Context, imageBase64 string) ([]ExtractedTx, error) {
	prompt := `Haal banktransacties uit dit afschrift (ABN AMRO of ING).
Geef ALLEEN JSON terug: [{"date":"YYYY-MM-DD","description":"...","amount":-12.34}]
Negatieve amount = uitgave, positieve = inkomst. Geen markdown.`

	body := map[string]any{
		"model":  c.visionModel,
		"prompt": prompt,
		"images": []string{imageBase64},
		"stream": false,
		"options": map[string]any{
			"temperature": 0,
		},
	}
	raw, err := c.post(ctx, "/api/generate", body)
	if err != nil {
		return nil, err
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	text := strings.TrimSpace(out.Response)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array in ollama response")
	}
	var items []ExtractedTx
	if err := json.Unmarshal([]byte(text[start:end+1]), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama %s: %s", res.Status, string(raw))
	}
	return raw, nil
}

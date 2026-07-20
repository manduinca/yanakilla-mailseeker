package zinc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL  string
	user     string
	password string
	http     *http.Client
}

func New(baseURL, user, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		user:     user,
		password: password,
		http: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

type bulkPayload struct {
	Index   string `json:"index"`
	Records []any  `json:"records"`
}

func (c *Client) EnsureIndex(name string) error {
	body := map[string]any{
		"name":         name,
		"storage_type": "disk",
		"mappings": map[string]any{
			"properties": map[string]any{
				"message_id": textField(false),
				"date":       textField(false),
				"from":       textField(true),
				"to":         textField(true),
				"cc":         textField(true),
				"bcc":        textField(false),
				"subject":    textField(true),
				"folder":     textField(false),
				"content":    textField(true),
			},
		},
	}

	resp, err := c.do(http.MethodPost, "/api/index", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return errorFrom(resp)
}

func textField(highlightable bool) map[string]any {
	return map[string]any{
		"type":          "text",
		"index":         true,
		"store":         true,
		"sortable":      false,
		"aggregatable":  false,
		"highlightable": highlightable,
	}
}

func (c *Client) Bulk(index string, records []any) error {
	if len(records) == 0 {
		return nil
	}

	resp, err := c.do(http.MethodPost, "/api/_bulkv2", bulkPayload{Index: index, Records: records})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errorFrom(resp)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

type SearchQuery struct {
	Term       string `json:"term"`
	Field      string `json:"field,omitempty"`
	StartTime  string `json:"start_time,omitempty"`
	EndTime    string `json:"end_time,omitempty"`
}

type SearchRequest struct {
	SearchType string      `json:"search_type"`
	Query      SearchQuery `json:"query"`
	SortFields []string    `json:"sort_fields,omitempty"`
	From       int         `json:"from"`
	MaxResults int         `json:"max_results"`
	Source     []string    `json:"_source"`
}

type Hit struct {
	Index     string          `json:"_index"`
	ID        string          `json:"_id"`
	Score     float64         `json:"_score"`
	Timestamp string          `json:"@timestamp"`
	Source    json.RawMessage `json:"_source"`
}

type SearchResponse struct {
	Took int  `json:"took"`
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []Hit   `json:"hits"`
	} `json:"hits"`
}

func (c *Client) Search(index string, req SearchRequest) (*SearchResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/"+index+"/_search", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorFrom(resp)
	}

	var out SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(method, path string, payload any) (*http.Response, error) {
	var buf bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.password)
	req.Header.Set("Content-Type", "application/json")

	return c.http.Do(req)
}

func errorFrom(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("zinc: %s: %s", resp.Status, bytes.TrimSpace(b))
}

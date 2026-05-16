package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"worker-service/internal/model"
)

type Client struct {
	url        string
	httpClient *http.Client
}

func New(url string) *Client {
	return &Client{
		url:        url,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SendEmail(ctx context.Context, req model.SendEmailRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode sender request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sender request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send email request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("sender returned %d: %s", resp.StatusCode, string(data))
	}

	return nil
}

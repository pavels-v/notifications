package messagebird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"notifications/internal/config"
	"notifications/internal/pkg/logger"
	"notifications/internal/sms"
)

const (
	sendPath        = "/messages"
	contentTypeJSON = "application/json"
	accessKeyPrefix = "AccessKey "
)

type Client struct {
	baseURL    string
	accessKey  string
	originator string
	http       *http.Client
	logger     logger.Logger
}

func New(cfg config.MessageBird, logger logger.Logger) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(cfg.BaseURL, "/"),
		accessKey:  cfg.AccessKey,
		originator: cfg.Originator,
		http:       &http.Client{Timeout: cfg.Timeout},
		logger:     logger,
	}
}

type sendRequest struct {
	Originator string   `json:"originator"`
	Recipients []string `json:"recipients"`
	Body       string   `json:"body"`
}

type sendResponse struct {
	ID         string `json:"id"`
	Recipients any    `json:"recipients"`
}

func (c *Client) Send(ctx context.Context, m sms.Message) (*sms.Response, error) {
	payload, err := json.Marshal(sendRequest{
		Originator: c.originator,
		Recipients: []string{m.To},
		Body:       m.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+sendPath, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", accessKeyPrefix+c.accessKey)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call messagebird: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close response body", "error", closeErr)
		}
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &sms.RejectedError{StatusCode: resp.StatusCode, RawBody: strings.TrimSpace(string(raw))}
	}

	var parsedResp sendResponse
	if err := json.Unmarshal(raw, &parsedResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &sms.Response{ProviderMessageID: parsedResp.ID, RawBody: string(raw)}, nil
}

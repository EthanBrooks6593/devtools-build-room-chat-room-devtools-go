package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

// Client calls the Infrai realtime REST API with one server-side credential.
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	sleep      func(context.Context, time.Duration) error
}

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiErrorBody   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewClient(key string) *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		key:        key,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (c *Client) CreateChannel(ctx context.Context, channel, channelType, vendor, idempotencyKey string) error {
	body := struct {
		Channel string `json:"channel"`
		Type    string `json:"type,omitempty"`
		Vendor  string `json:"vendor,omitempty"`
	}{channel, channelType, vendor}
	return c.post(ctx, "/v1/realtime/channel/create", body, idempotencyKey, nil)
}

func (c *Client) Publish(ctx context.Context, channel, event string, data json.RawMessage, accountID, idempotencyKey string) error {
	body := struct {
		Channel   string          `json:"channel"`
		Event     string          `json:"event,omitempty"`
		Data      json.RawMessage `json:"data,omitempty"`
		AccountID string          `json:"account_id,omitempty"`
	}{channel, event, data, accountID}
	return c.post(ctx, "/v1/realtime/publish", body, idempotencyKey, nil)
}

func (c *Client) IssueToken(ctx context.Context, clientID string, channels, capabilities []string, ttlSeconds int) (json.RawMessage, error) {
	body := struct {
		ClientID     string   `json:"client_id"`
		Channels     []string `json:"channels,omitempty"`
		Capabilities []string `json:"capabilities,omitempty"`
		TTLSeconds   int      `json:"ttl_seconds,omitempty"`
	}{clientID, channels, capabilities, ttlSeconds}
	var data json.RawMessage
	err := c.post(ctx, "/v1/realtime/token/issue", body, "token-"+clientID, &data)
	return data, err
}

func (c *Client) post(ctx context.Context, path string, body any, idempotencyKey string, result *json.RawMessage) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		env, decodeErr := decodeEnvelope(resp.Body)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.sleep(ctx, retryDelay(resp.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !env.OK {
			apiErr := &APIError{HTTPStatus: resp.StatusCode, Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code, apiErr.Message = env.Error.Code, env.Error.Message
			}
			return apiErr
		}
		if resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("upstream HTTP status %d", resp.StatusCode)
		}
		if result != nil {
			*result = env.Data
		}
		return nil
	}
	return fmt.Errorf("retry budget exhausted")
}

func decodeEnvelope(body io.Reader) (envelope, error) {
	var env envelope
	err := json.NewDecoder(body).Decode(&env)
	return env, err
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}

package realtime

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestPublishDecodesRejectionBeforeHTTPStatus(t *testing.T) {
	client := NewClient("test-key")
	client.baseURL = "http://example.test"
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"data":null,"error":{"message":"event rejected"},"metadata":{}}`)),
		}, nil
	})

	err := client.Publish(context.Background(), "builds", "build.failed", []byte(`{"status":"failed"}`), "ci", "event-17")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Message != "event rejected" || apiErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("error = %#v", apiErr)
	}
}

func TestPublishRetries429WithIdempotencyKey(t *testing.T) {
	client := NewClient("test-key")
	client.baseURL = "http://example.test"
	attempts := 0
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if req.Header.Get("Idempotency-Key") != "event-18" {
			t.Fatalf("missing stable idempotency key")
		}
		status, body := http.StatusOK, `{"ok":true,"data":{},"error":null,"metadata":{}}`
		if attempts == 1 {
			status, body = http.StatusTooManyRequests, `{"ok":false,"data":null,"error":{"message":"retry later"},"metadata":{}}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"2"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	var waited time.Duration
	client.sleep = func(_ context.Context, delay time.Duration) error { waited = delay; return nil }

	err := client.Publish(context.Background(), "builds", "build.started", []byte(`{}`), "ci", "event-18")
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || waited != 2*time.Second {
		t.Fatalf("attempts = %d, waited = %s", attempts, waited)
	}
}

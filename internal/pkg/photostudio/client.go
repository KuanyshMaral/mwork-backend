package photostudio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
)

const defaultTimeout = 10 * time.Second

// Client represents PhotoStudio HTTP client.
type Client struct {
	baseURL string
	token   string
	ua      string
	http    *http.Client
}

// SyncUserPayload represents user sync payload.
type SyncUserPayload struct {
	MWorkUserID string `json:"mwork_user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// NewClient creates a new PhotoStudio client.
func NewClient(baseURL, token string, timeout time.Duration, ua string) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		ua:      ua,
		http: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// SyncUser sends a user sync request to PhotoStudio.
func (c *Client) SyncUser(ctx context.Context, p SyncUserPayload) error {
	if c == nil || c.http == nil {
		return fmt.Errorf("photostudio sync request error: client is nil")
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return fmt.Errorf("photostudio sync config error: base_url is empty")
	}
	if strings.TrimSpace(c.token) == "" {
		return fmt.Errorf("photostudio sync config error: token is empty")
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("photostudio sync request error: %w", err)
	}

	url := c.baseURL + "/internal/mwork/users/sync"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("photostudio sync request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if c.ua != "" {
		req.Header.Set("User-Agent", c.ua)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return classifyRequestError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("photostudio sync http error: status=%d body=<failed to read body: %v>", resp.StatusCode, readErr)
	}

	return fmt.Errorf("photostudio sync http error: status=%d body=%s", resp.StatusCode, string(body))
}

func classifyRequestError(ctx context.Context, err error) error {
	if isTimeoutError(ctx, err) {
		return fmt.Errorf("photostudio sync timeout: %w", err)
	}
	if isNetworkError(err) {
		return fmt.Errorf("photostudio sync network error: %w", err)
	}
	return fmt.Errorf("photostudio sync request error: %w", err)
}

func isTimeoutError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}

	return false
}

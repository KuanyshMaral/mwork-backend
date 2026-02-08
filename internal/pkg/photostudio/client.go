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

	"github.com/google/uuid"
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

// BookingRequest represents booking creation request.
type BookingRequest struct {
	RoomID    int64     `json:"room_id"`
	StudioID  int64     `json:"studio_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Notes     string    `json:"notes,omitempty"`
}

// BookingResponse represents booking creation response.
type BookingResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Booking struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"booking"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Studio represents studio information.
type Studio struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	Rating      float64 `json:"rating"`
}

// StudiosResponse represents studios list response.
type StudiosResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Studios    []Studio `json:"studios"`
		Pagination struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"data"`
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

// CreateBooking creates a booking at PhotoStudio.
// CRITICAL: Must send X-MWork-User-ID header for user identification.
func (c *Client) CreateBooking(ctx context.Context, userID uuid.UUID, req BookingRequest) (*BookingResponse, error) {
	if c == nil || c.http == nil {
		return nil, fmt.Errorf("photostudio booking error: client is nil")
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, fmt.Errorf("photostudio booking error: base_url is empty")
	}
	if strings.TrimSpace(c.token) == "" {
		return nil, fmt.Errorf("photostudio booking error: token is empty")
	}
	if userID == uuid.Nil {
		return nil, fmt.Errorf("photostudio booking error: userID is nil")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("photostudio booking marshal error: %w", err)
	}

	url := c.baseURL + "/internal/mwork/bookings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("photostudio booking request error: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("X-MWork-User-ID", userID.String()) // CRITICAL: User ID mapping
	if c.ua != "" {
		httpReq.Header.Set("User-Agent", c.ua)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, classifyRequestError(ctx, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("photostudio booking read error: %w", readErr)
	}

	var result BookingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("photostudio booking parse error: status=%d body=%s err=%w", resp.StatusCode, string(body), err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if result.Error != nil {
			return &result, fmt.Errorf("photostudio booking failed: status=%d code=%s message=%s", resp.StatusCode, result.Error.Code, result.Error.Message)
		}
		return &result, fmt.Errorf("photostudio booking failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return &result, nil
}

// GetStudios retrieves studios list from PhotoStudio.
func (c *Client) GetStudios(ctx context.Context, city string, page, limit int) (*StudiosResponse, error) {
	if c == nil || c.http == nil {
		return nil, fmt.Errorf("photostudio studios error: client is nil")
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, fmt.Errorf("photostudio studios error: base_url is empty")
	}

	// Build query params
	params := url.Values{}
	if city != "" {
		params.Set("city", city)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	url := c.baseURL + "/api/v1/studios"
	if len(params) > 0 {
		url += "?" + params.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("photostudio studios request error: %w", err)
	}

	// Optional: Add auth token for internal requests
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.ua != "" {
		httpReq.Header.Set("User-Agent", c.ua)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, classifyRequestError(ctx, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("photostudio studios read error: %w", readErr)
	}

	var result StudiosResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("photostudio studios parse error: status=%d body=%s err=%w", resp.StatusCode, string(body), err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("photostudio studios failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return &result, nil
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

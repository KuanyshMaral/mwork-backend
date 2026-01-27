package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FCMConfig holds Firebase Cloud Messaging configuration
type FCMConfig struct {
	ServerKey string
	ProjectID string
}

// FCMClient sends push notifications via Firebase Cloud Messaging
type FCMClient struct {
	config     FCMConfig
	httpClient *http.Client
}

// NewFCMClient creates a new FCM client
func NewFCMClient(config FCMConfig) *FCMClient {
	return &FCMClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PushMessage represents a push notification
type PushMessage struct {
	Token string // Device token
	Title string
	Body  string
	Data  map[string]string // Custom data
	Badge int               // Badge count (iOS)
}

// FCMRequest represents the FCM HTTP v1 API request
type FCMRequest struct {
	Message FCMMessage `json:"message"`
}

type FCMMessage struct {
	Token        string            `json:"token"`
	Notification *FCMNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *FCMAndroid       `json:"android,omitempty"`
	Webpush      *FCMWebpush       `json:"webpush,omitempty"`
	APNS         *FCMAPNS          `json:"apns,omitempty"`
}

type FCMNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type FCMAndroid struct {
	Priority     string                  `json:"priority,omitempty"` // "high" or "normal"
	Notification *FCMAndroidNotification `json:"notification,omitempty"`
}

type FCMAndroidNotification struct {
	ClickAction string `json:"click_action,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
}

type FCMWebpush struct {
	Notification *FCMWebpushNotification `json:"notification,omitempty"`
	FCMOptions   *FCMOptions             `json:"fcm_options,omitempty"`
}

type FCMWebpushNotification struct {
	Icon string `json:"icon,omitempty"`
}

type FCMOptions struct {
	Link string `json:"link,omitempty"`
}

type FCMAPNS struct {
	Payload *APNSPayload `json:"payload,omitempty"`
}

type APNSPayload struct {
	Aps *APNSAps `json:"aps,omitempty"`
}

type APNSAps struct {
	Badge int    `json:"badge,omitempty"`
	Sound string `json:"sound,omitempty"`
}

// Send sends a push notification
func (c *FCMClient) Send(ctx context.Context, msg *PushMessage) error {
	request := FCMRequest{
		Message: FCMMessage{
			Token: msg.Token,
			Notification: &FCMNotification{
				Title: msg.Title,
				Body:  msg.Body,
			},
			Data: msg.Data,
			Android: &FCMAndroid{
				Priority: "high",
				Notification: &FCMAndroidNotification{
					ClickAction: "FLUTTER_NOTIFICATION_CLICK",
					Color:       "#a855f7",
				},
			},
			Webpush: &FCMWebpush{
				Notification: &FCMWebpushNotification{
					Icon: "/icon-192.png",
				},
			},
		},
	}

	// Add badge for iOS
	if msg.Badge > 0 {
		request.Message.APNS = &FCMAPNS{
			Payload: &APNSPayload{
				Aps: &APNSAps{
					Badge: msg.Badge,
					Sound: "default",
				},
			},
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM request: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", c.config.ProjectID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.ServerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send FCM request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("FCM returned status %d", resp.StatusCode)
	}

	return nil
}

// SendMultiple sends push notification to multiple tokens
func (c *FCMClient) SendMultiple(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	for _, token := range tokens {
		msg := &PushMessage{
			Token: token,
			Title: title,
			Body:  body,
			Data:  data,
		}
		// Fire and forget, log errors but don't fail
		go func(m *PushMessage) {
			if err := c.Send(context.Background(), m); err != nil {
				// Log error but continue
			}
		}(msg)
	}
	return nil
}

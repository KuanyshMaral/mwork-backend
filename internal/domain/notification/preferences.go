package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ChannelSettings represents notification settings per channel
type ChannelSettings struct {
	InApp bool `json:"in_app"`
	Email bool `json:"email"`
	Push  bool `json:"push"`
}

// UserPreferences holds user notification preferences
type UserPreferences struct {
	ID     uuid.UUID `db:"id" json:"id"`
	UserID uuid.UUID `db:"user_id" json:"user_id"`

	// Global toggles
	EmailEnabled bool `db:"email_enabled" json:"email_enabled"`
	PushEnabled  bool `db:"push_enabled" json:"push_enabled"`
	InAppEnabled bool `db:"in_app_enabled" json:"in_app_enabled"`

	// Per-type settings (stored as JSONB)
	NewResponseChannels      json.RawMessage `db:"new_response_channels" json:"new_response_channels"`
	ResponseAcceptedChannels json.RawMessage `db:"response_accepted_channels" json:"response_accepted_channels"`
	ResponseRejectedChannels json.RawMessage `db:"response_rejected_channels" json:"response_rejected_channels"`
	NewMessageChannels       json.RawMessage `db:"new_message_channels" json:"new_message_channels"`
	ProfileViewedChannels    json.RawMessage `db:"profile_viewed_channels" json:"profile_viewed_channels"`
	CastingExpiringChannels  json.RawMessage `db:"casting_expiring_channels" json:"casting_expiring_channels"`

	// Digest settings
	DigestEnabled   bool   `db:"digest_enabled" json:"digest_enabled"`
	DigestFrequency string `db:"digest_frequency" json:"digest_frequency"`
}

// DeviceToken represents a push notification device token
type DeviceToken struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Token      string    `db:"token" json:"token"`
	Platform   string    `db:"platform" json:"platform"` // web, android, ios
	DeviceName string    `db:"device_name" json:"device_name,omitempty"`
	IsActive   bool      `db:"is_active" json:"is_active"`
}

// NotificationGroup for batching notifications
type NotificationGroup struct {
	ID                  uuid.UUID       `db:"id" json:"id"`
	UserID              uuid.UUID       `db:"user_id" json:"user_id"`
	Type                Type            `db:"type" json:"type"`
	Count               int             `db:"count" json:"count"`
	SummaryData         json.RawMessage `db:"summary_data" json:"summary_data"`
	FirstNotificationID uuid.UUID       `db:"first_notification_id" json:"first_notification_id"`
	SummarySent         bool            `db:"summary_sent" json:"summary_sent"`
}

// PreferencesRepository handles preferences data access
type PreferencesRepository struct {
	db *sqlx.DB
}

// NewPreferencesRepository creates preferences repository
func NewPreferencesRepository(db *sqlx.DB) *PreferencesRepository {
	return &PreferencesRepository{db: db}
}

// GetByUserID gets preferences for user (creates default if not exists)
func (r *PreferencesRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	var prefs UserPreferences
	err := r.db.GetContext(ctx, &prefs, `
		SELECT * FROM user_notification_preferences WHERE user_id = $1
	`, userID)

	if err != nil {
		// Create default preferences
		prefs = UserPreferences{
			ID:              uuid.New(),
			UserID:          userID,
			EmailEnabled:    true,
			PushEnabled:     true,
			InAppEnabled:    true,
			DigestEnabled:   true,
			DigestFrequency: "weekly",
		}

		// Set default channel settings
		defaultOn := []byte(`{"in_app": true, "email": true, "push": true}`)
		defaultOff := []byte(`{"in_app": true, "email": false, "push": false}`)

		prefs.NewResponseChannels = defaultOn
		prefs.ResponseAcceptedChannels = defaultOn
		prefs.ResponseRejectedChannels = []byte(`{"in_app": true, "email": true, "push": false}`)
		prefs.NewMessageChannels = []byte(`{"in_app": true, "email": false, "push": true}`)
		prefs.ProfileViewedChannels = defaultOff
		prefs.CastingExpiringChannels = []byte(`{"in_app": true, "email": true, "push": false}`)

		_, err = r.db.ExecContext(ctx, `
			INSERT INTO user_notification_preferences (
				id, user_id, email_enabled, push_enabled, in_app_enabled,
				new_response_channels, response_accepted_channels, response_rejected_channels,
				new_message_channels, profile_viewed_channels, casting_expiring_channels,
				digest_enabled, digest_frequency
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, prefs.ID, prefs.UserID, prefs.EmailEnabled, prefs.PushEnabled, prefs.InAppEnabled,
			prefs.NewResponseChannels, prefs.ResponseAcceptedChannels, prefs.ResponseRejectedChannels,
			prefs.NewMessageChannels, prefs.ProfileViewedChannels, prefs.CastingExpiringChannels,
			prefs.DigestEnabled, prefs.DigestFrequency)

		if err != nil {
			return nil, err
		}
	}

	return &prefs, nil
}

// Update updates user preferences
func (r *PreferencesRepository) Update(ctx context.Context, prefs *UserPreferences) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE user_notification_preferences SET
			email_enabled = $2,
			push_enabled = $3,
			in_app_enabled = $4,
			new_response_channels = $5,
			response_accepted_channels = $6,
			response_rejected_channels = $7,
			new_message_channels = $8,
			profile_viewed_channels = $9,
			casting_expiring_channels = $10,
			digest_enabled = $11,
			digest_frequency = $12,
			updated_at = NOW()
		WHERE user_id = $1
	`, prefs.UserID, prefs.EmailEnabled, prefs.PushEnabled, prefs.InAppEnabled,
		prefs.NewResponseChannels, prefs.ResponseAcceptedChannels, prefs.ResponseRejectedChannels,
		prefs.NewMessageChannels, prefs.ProfileViewedChannels, prefs.CastingExpiringChannels,
		prefs.DigestEnabled, prefs.DigestFrequency)
	return err
}

// GetChannelsForType returns enabled channels for a notification type
func (prefs *UserPreferences) GetChannelsForType(notifType Type) ChannelSettings {
	var raw json.RawMessage

	switch notifType {
	case TypeNewResponse:
		raw = prefs.NewResponseChannels
	case TypeResponseAccepted:
		raw = prefs.ResponseAcceptedChannels
	case TypeResponseRejected:
		raw = prefs.ResponseRejectedChannels
	case TypeNewMessage:
		raw = prefs.NewMessageChannels
	case TypeProfileViewed:
		raw = prefs.ProfileViewedChannels
	case TypeCastingExpiring:
		raw = prefs.CastingExpiringChannels
	default:
		return ChannelSettings{InApp: true, Email: false, Push: false}
	}

	var settings ChannelSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return ChannelSettings{InApp: true, Email: false, Push: false}
	}

	// Apply global toggles
	if !prefs.InAppEnabled {
		settings.InApp = false
	}
	if !prefs.EmailEnabled {
		settings.Email = false
	}
	if !prefs.PushEnabled {
		settings.Push = false
	}

	return settings
}

// DeviceTokenRepository handles device tokens
type DeviceTokenRepository struct {
	db *sqlx.DB
}

// NewDeviceTokenRepository creates device token repository
func NewDeviceTokenRepository(db *sqlx.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{db: db}
}

// Save saves or updates a device token
func (r *DeviceTokenRepository) Save(ctx context.Context, token *DeviceToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO device_tokens (id, user_id, token, platform, device_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (token) DO UPDATE SET
			user_id = $2,
			is_active = true,
			last_used_at = NOW()
	`, token.ID, token.UserID, token.Token, token.Platform, token.DeviceName, true)
	return err
}

// GetActiveByUserID gets active device tokens for user
func (r *DeviceTokenRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var tokens []string
	err := r.db.SelectContext(ctx, &tokens, `
		SELECT token FROM device_tokens 
		WHERE user_id = $1 AND is_active = true
	`, userID)
	return tokens, err
}

// Deactivate deactivates a device token
func (r *DeviceTokenRepository) Deactivate(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE device_tokens SET is_active = false WHERE token = $1
	`, token)
	return err
}

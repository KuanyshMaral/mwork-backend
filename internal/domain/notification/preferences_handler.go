package notification

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// PreferencesHandler handles notification preferences API
type PreferencesHandler struct {
	prefsRepo  *PreferencesRepository
	deviceRepo *DeviceTokenRepository
}

// NewPreferencesHandler creates preferences handler
func NewPreferencesHandler(prefsRepo *PreferencesRepository, deviceRepo *DeviceTokenRepository) *PreferencesHandler {
	return &PreferencesHandler{
		prefsRepo:  prefsRepo,
		deviceRepo: deviceRepo,
	}
}

// Routes returns preferences routes
func (h *PreferencesHandler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", h.GetPreferences)
	r.Put("/", h.UpdatePreferences)
	r.Post("/device", h.RegisterDevice)
	r.Delete("/device/{token}", h.UnregisterDevice)

	return r
}

// GetPreferences handles GET /api/v1/notifications/preferences
// @Summary Получить настройки уведомлений
// @Tags Notification
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401,500 {object} response.Response
// @Router /notifications/preferences [get]
func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	prefs, err := h.prefsRepo.GetByUserID(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, prefsToResponse(prefs))
}

// UpdatePreferencesRequest for updating preferences
type UpdatePreferencesRequest struct {
	EmailEnabled *bool `json:"email_enabled"`
	PushEnabled  *bool `json:"push_enabled"`
	InAppEnabled *bool `json:"in_app_enabled"`

	NewResponseChannels      *ChannelSettings `json:"new_response_channels"`
	ResponseAcceptedChannels *ChannelSettings `json:"response_accepted_channels"`
	ResponseRejectedChannels *ChannelSettings `json:"response_rejected_channels"`
	NewMessageChannels       *ChannelSettings `json:"new_message_channels"`
	ProfileViewedChannels    *ChannelSettings `json:"profile_viewed_channels"`
	CastingExpiringChannels  *ChannelSettings `json:"casting_expiring_channels"`

	DigestEnabled   *bool   `json:"digest_enabled"`
	DigestFrequency *string `json:"digest_frequency"`
}

// UpdatePreferences handles PUT /api/v1/notifications/preferences
// @Summary Обновить настройки уведомлений
// @Tags Notification
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdatePreferencesRequest true "Настройки"
// @Success 200 {object} response.Response
// @Failure 400,401,500 {object} response.Response
// @Router /notifications/preferences [put]
func (h *PreferencesHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	var req UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Get existing preferences
	prefs, err := h.prefsRepo.GetByUserID(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Apply updates
	if req.EmailEnabled != nil {
		prefs.EmailEnabled = *req.EmailEnabled
	}
	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.InAppEnabled != nil {
		prefs.InAppEnabled = *req.InAppEnabled
	}
	if req.DigestEnabled != nil {
		prefs.DigestEnabled = *req.DigestEnabled
	}
	if req.DigestFrequency != nil {
		prefs.DigestFrequency = *req.DigestFrequency
	}

	// Update channel settings
	if req.NewResponseChannels != nil {
		prefs.NewResponseChannels, _ = json.Marshal(req.NewResponseChannels)
	}
	if req.ResponseAcceptedChannels != nil {
		prefs.ResponseAcceptedChannels, _ = json.Marshal(req.ResponseAcceptedChannels)
	}
	if req.ResponseRejectedChannels != nil {
		prefs.ResponseRejectedChannels, _ = json.Marshal(req.ResponseRejectedChannels)
	}
	if req.NewMessageChannels != nil {
		prefs.NewMessageChannels, _ = json.Marshal(req.NewMessageChannels)
	}
	if req.ProfileViewedChannels != nil {
		prefs.ProfileViewedChannels, _ = json.Marshal(req.ProfileViewedChannels)
	}
	if req.CastingExpiringChannels != nil {
		prefs.CastingExpiringChannels, _ = json.Marshal(req.CastingExpiringChannels)
	}

	if err := h.prefsRepo.Update(r.Context(), prefs); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, prefsToResponse(prefs))
}

// RegisterDeviceRequest for registering device token
type RegisterDeviceRequest struct {
	Token      string `json:"token" validate:"required"`
	Platform   string `json:"platform" validate:"required,oneof=web android ios"`
	DeviceName string `json:"device_name"`
}

// RegisterDevice handles POST /api/v1/notifications/preferences/device
// @Summary Зарегистрировать push-токен устройства
// @Tags Notification
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RegisterDeviceRequest true "Данные устройства"
// @Success 200 {object} response.Response
// @Failure 400,401,500 {object} response.Response
// @Router /notifications/preferences/device [post]
func (h *PreferencesHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.Token == "" || req.Platform == "" {
		response.BadRequest(w, "Token and platform are required")
		return
	}

	token := &DeviceToken{
		ID:         uuid.New(),
		UserID:     userID,
		Token:      req.Token,
		Platform:   req.Platform,
		DeviceName: req.DeviceName,
		IsActive:   true,
	}

	if err := h.deviceRepo.Save(r.Context(), token); err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, map[string]string{"status": "registered"})
}

// UnregisterDevice handles DELETE /api/v1/notifications/preferences/device/{token}
// @Summary Удалить push-токен устройства
// @Tags Notification
// @Produce json
// @Security BearerAuth
// @Param token path string true "FCM токен"
// @Success 200 {object} response.Response
// @Failure 400,401,500 {object} response.Response
// @Router /notifications/preferences/device/{token} [delete]
func (h *PreferencesHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		response.BadRequest(w, "Token is required")
		return
	}

	if err := h.deviceRepo.Deactivate(r.Context(), token); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "unregistered"})
}

// PreferencesResponse for API response
type PreferencesResponse struct {
	EmailEnabled bool `json:"email_enabled"`
	PushEnabled  bool `json:"push_enabled"`
	InAppEnabled bool `json:"in_app_enabled"`

	NewResponseChannels      ChannelSettings `json:"new_response_channels"`
	ResponseAcceptedChannels ChannelSettings `json:"response_accepted_channels"`
	ResponseRejectedChannels ChannelSettings `json:"response_rejected_channels"`
	NewMessageChannels       ChannelSettings `json:"new_message_channels"`
	ProfileViewedChannels    ChannelSettings `json:"profile_viewed_channels"`
	CastingExpiringChannels  ChannelSettings `json:"casting_expiring_channels"`

	DigestEnabled   bool   `json:"digest_enabled"`
	DigestFrequency string `json:"digest_frequency"`
}

func prefsToResponse(p *UserPreferences) *PreferencesResponse {
	resp := &PreferencesResponse{
		EmailEnabled:    p.EmailEnabled,
		PushEnabled:     p.PushEnabled,
		InAppEnabled:    p.InAppEnabled,
		DigestEnabled:   p.DigestEnabled,
		DigestFrequency: p.DigestFrequency,
	}

	json.Unmarshal(p.NewResponseChannels, &resp.NewResponseChannels)
	json.Unmarshal(p.ResponseAcceptedChannels, &resp.ResponseAcceptedChannels)
	json.Unmarshal(p.ResponseRejectedChannels, &resp.ResponseRejectedChannels)
	json.Unmarshal(p.NewMessageChannels, &resp.NewMessageChannels)
	json.Unmarshal(p.ProfileViewedChannels, &resp.ProfileViewedChannels)
	json.Unmarshal(p.CastingExpiringChannels, &resp.CastingExpiringChannels)

	return resp
}

// Helper to get user ID from context
func getUserIDFromContext(ctx context.Context) uuid.UUID {
	return middleware.GetUserID(ctx)
}

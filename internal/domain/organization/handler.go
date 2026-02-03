package organization

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles organization HTTP requests
type Handler struct {
	service   *Service
	validator *validator.Validate
}

// NewHandler creates organization handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

// Get returns organization by ID
// GET /organizations/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	org, err := h.service.GetByID(r.Context(), id)
	if err != nil || org == nil {
		response.NotFound(w, "organization not found")
		return
	}

	response.OK(w, ToResponse(org))
}

// GetCastings returns public castings from an organization
// GET /organizations/{id}/castings
func (h *Handler) GetCastings(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	// Check if org exists
	org, err := h.service.GetByID(r.Context(), id)
	if err != nil || org == nil {
		response.NotFound(w, "organization not found")
		return
	}

	// TODO: Implement casting listing by organization
	// For now, return empty list
	response.OK(w, map[string]interface{}{
		"items": []interface{}{},
		"total": 0,
	})
}

// GetMembers returns organization members
// GET /organizations/{id}/members
func (h *Handler) GetMembers(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	// Check if user is a member
	member, _ := h.service.repo.GetMemberByUserID(r.Context(), id, userID)
	if member == nil {
		response.Forbidden(w, "not a member of this organization")
		return
	}

	members, err := h.service.GetMembers(r.Context(), id)
	if err != nil {
		response.InternalError(w)
		return
	}

	var items []*MemberResponse
	for _, m := range members {
		items = append(items, ToMemberResponse(m))
	}

	response.OK(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// InviteMember invites a new member to organization
// POST /organizations/{id}/members/invite
func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	member, err := h.service.AddMember(r.Context(), id, userID, req.UserID, req.Role)
	if err != nil {
		switch err {
		case ErrNotAuthorized:
			response.Forbidden(w, "not authorized to add members")
		case ErrMemberAlreadyExists:
			response.Conflict(w, "user is already a member")
		case ErrUserNotFound:
			response.NotFound(w, "user not found")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, ToMemberResponse(member))
}

// UpdateMemberRole updates a member's role
// PATCH /organizations/{id}/members/{memberId}
func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "memberId"))
	if err != nil {
		response.BadRequest(w, "invalid member id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if err := h.service.UpdateMemberRole(r.Context(), id, userID, memberID, req.Role); err != nil {
		switch err {
		case ErrNotAuthorized:
			response.Forbidden(w, "not authorized to update member roles")
		case ErrMemberNotFound:
			response.NotFound(w, "member not found")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": "updated"})
}

// RemoveMember removes a member from organization
// DELETE /organizations/{id}/members/{memberId}
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "memberId"))
	if err != nil {
		response.BadRequest(w, "invalid member id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	if err := h.service.RemoveMember(r.Context(), id, userID, memberID); err != nil {
		switch err {
		case ErrNotAuthorized:
			response.Forbidden(w, "not authorized to remove members")
		case ErrMemberNotFound:
			response.NotFound(w, "member not found")
		case ErrCannotRemoveOwner:
			response.BadRequest(w, "cannot remove organization owner")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": "removed"})
}

// Follow allows a user to follow an organization
// POST /organizations/{id}/follow
func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	if err := h.service.Follow(r.Context(), id, userID); err != nil {
		switch err {
		case ErrOrganizationNotFound:
			response.NotFound(w, "organization not found")
		case ErrAlreadyFollowing:
			response.Conflict(w, "already following this organization")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": "following"})
}

// Unfollow allows a user to unfollow an organization
// DELETE /organizations/{id}/follow
func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	if err := h.service.Unfollow(r.Context(), id, userID); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "unfollowed"})
}

// CheckFollowing checks if user is following an organization
// GET /organizations/{id}/follow/check
func (h *Handler) CheckFollowing(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid organization id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	isFollowing, err := h.service.CheckFollowing(r.Context(), id, userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{
		"is_following": isFollowing,
	})
}

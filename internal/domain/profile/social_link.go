package profile

import (
	"time"

	"github.com/google/uuid"
)

// SocialLink represents a social media link for a profile
type SocialLink struct {
	ID         uuid.UUID `db:"id"`
	ProfileID  uuid.UUID `db:"profile_id"`
	Platform   string    `db:"platform"`
	URL        string    `db:"url"`
	Username   string    `db:"username"`
	IsVerified bool      `db:"is_verified"`
	CreatedAt  time.Time `db:"created_at"`
}

// ValidPlatforms lists allowed social media platforms
var ValidPlatforms = []string{
	"instagram",
	"tiktok",
	"facebook",
	"twitter",
	"youtube",
	"telegram",
	"linkedin",
	"vk",
}

// IsValidPlatform checks if platform is allowed
func IsValidPlatform(platform string) bool {
	for _, p := range ValidPlatforms {
		if p == platform {
			return true
		}
	}
	return false
}

// SocialLinkRequest for adding/updating social links
type SocialLinkRequest struct {
	Platform string `json:"platform" validate:"required,oneof=instagram tiktok facebook twitter youtube telegram linkedin vk"`
	URL      string `json:"url" validate:"required,url,max=500"`
	Username string `json:"username" validate:"omitempty,max=100"`
}

// ToEntity converts request to entity
func (r *SocialLinkRequest) ToEntity(profileID uuid.UUID) *SocialLink {
	return &SocialLink{
		ID:        uuid.New(),
		ProfileID: profileID,
		Platform:  r.Platform,
		URL:       r.URL,
		Username:  r.Username,
		CreatedAt: time.Now(),
	}
}

// ToResponse converts entity to response
func (s *SocialLink) ToResponse() SocialLinkResponse {
	return SocialLinkResponse{
		Platform:   s.Platform,
		URL:        s.URL,
		Username:   s.Username,
		IsVerified: s.IsVerified,
	}
}

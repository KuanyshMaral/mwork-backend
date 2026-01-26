package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Stats represents dashboard statistics
type Stats struct {
	// Common
	ProfileViews  int `json:"profile_views"`
	MessagesCount int `json:"messages_count"`
	NewMessages   int `json:"new_messages"`

	// Model specific
	ApplicationsCount int `json:"applications_count,omitempty"`
	AcceptedCount     int `json:"accepted_count,omitempty"`

	// Employer specific
	CastingsCount  int `json:"castings_count,omitempty"`
	TotalResponses int `json:"total_responses,omitempty"`
	ActiveCastings int `json:"active_castings,omitempty"`
	TotalViews     int `json:"total_views,omitempty"`
}

// Service provides dashboard statistics
type Service struct {
	db *sqlx.DB
}

// NewService creates dashboard service
func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// GetModelStats returns statistics for a model user
func (s *Service) GetModelStats(ctx context.Context, userID uuid.UUID) (*Stats, error) {
	stats := &Stats{}
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// Get profile ID
	var profileID uuid.UUID
	err := s.db.GetContext(ctx, &profileID,
		`SELECT id FROM model_profiles WHERE user_id = $1`, userID)
	if err != nil {
		return stats, nil // Return empty stats if no profile
	}

	// Profile views (last 30 days)
	_ = s.db.GetContext(ctx, &stats.ProfileViews,
		`SELECT COUNT(*) FROM profile_views 
		 WHERE profile_id = $1 AND viewed_at > $2`,
		profileID, thirtyDaysAgo)

	// Total applications
	_ = s.db.GetContext(ctx, &stats.ApplicationsCount,
		`SELECT COUNT(*) FROM responses WHERE model_id = $1`,
		profileID)

	// Accepted applications
	_ = s.db.GetContext(ctx, &stats.AcceptedCount,
		`SELECT COUNT(*) FROM responses 
		 WHERE model_id = $1 AND status = 'accepted'`,
		profileID)

	// Total messages
	_ = s.db.GetContext(ctx, &stats.MessagesCount,
		`SELECT COUNT(*) FROM chat_messages cm
		 JOIN chat_rooms cr ON cm.room_id = cr.id
		 WHERE cr.participant_1_id = $1 OR cr.participant_2_id = $1`,
		userID)

	// Unread messages
	_ = s.db.GetContext(ctx, &stats.NewMessages,
		`SELECT COUNT(*) FROM chat_messages cm
		 JOIN chat_rooms cr ON cm.room_id = cr.id
		 WHERE (cr.participant_1_id = $1 OR cr.participant_2_id = $1)
		 AND cm.sender_id != $1 AND cm.is_read = false`,
		userID)

	return stats, nil
}

// GetEmployerStats returns statistics for an employer user
func (s *Service) GetEmployerStats(ctx context.Context, userID uuid.UUID) (*Stats, error) {
	stats := &Stats{}

	// Total castings
	_ = s.db.GetContext(ctx, &stats.CastingsCount,
		`SELECT COUNT(*) FROM castings WHERE creator_id = $1`,
		userID)

	// Active castings
	_ = s.db.GetContext(ctx, &stats.ActiveCastings,
		`SELECT COUNT(*) FROM castings 
		 WHERE creator_id = $1 AND status = 'active'`,
		userID)

	// Total responses on all castings
	_ = s.db.GetContext(ctx, &stats.TotalResponses,
		`SELECT COUNT(*) FROM responses r
		 JOIN castings c ON r.casting_id = c.id
		 WHERE c.creator_id = $1`,
		userID)

	// Total views on all castings
	_ = s.db.GetContext(ctx, &stats.TotalViews,
		`SELECT COALESCE(SUM(view_count), 0) FROM castings 
		 WHERE creator_id = $1`,
		userID)

	// Total messages
	_ = s.db.GetContext(ctx, &stats.MessagesCount,
		`SELECT COUNT(*) FROM chat_messages cm
		 JOIN chat_rooms cr ON cm.room_id = cr.id
		 WHERE cr.participant_1_id = $1 OR cr.participant_2_id = $1`,
		userID)

	// Unread messages
	_ = s.db.GetContext(ctx, &stats.NewMessages,
		`SELECT COUNT(*) FROM chat_messages cm
		 JOIN chat_rooms cr ON cm.room_id = cr.id
		 WHERE (cr.participant_1_id = $1 OR cr.participant_2_id = $1)
		 AND cm.sender_id != $1 AND cm.is_read = false`,
		userID)

	return stats, nil
}

package photostudio_booking

import "time"

// CreateBookingRequest represents booking creation request from frontend.
type CreateBookingRequest struct {
	StudioID  int64     `json:"studio_id" validate:"required,min=1"`
	RoomID    int64     `json:"room_id" validate:"required,min=1"`
	StartTime time.Time `json:"start_time" validate:"required"`
	EndTime   time.Time `json:"end_time" validate:"required"`
	Notes     string    `json:"notes"`
}

// BookingResponse represents booking response to frontend.
type BookingResponse struct {
	BookingID int64  `json:"booking_id"`
	Status    string `json:"status"`
}

// StudioListRequest represents studios list request.
type StudioListRequest struct {
	City  string `json:"city"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

// StudioResponse represents studio information.
type StudioResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	Rating      float64 `json:"rating"`
}

// StudiosListResponse represents studios list response.
type StudiosListResponse struct {
	Studios    []StudioResponse `json:"studios"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

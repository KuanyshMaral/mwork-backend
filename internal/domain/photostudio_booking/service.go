package photostudio_booking

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
)

// PhotoStudioClient interface for mocking in tests.
type PhotoStudioClient interface {
	CreateBooking(ctx context.Context, userID uuid.UUID, req photostudio.BookingRequest) (*photostudio.BookingResponse, error)
	GetStudios(ctx context.Context, city string, page, limit int) (*photostudio.StudiosResponse, error)
}

// Service handles PhotoStudio booking business logic.
type Service struct {
	client  *photostudio.Client
	enabled bool
}

// NewService creates a new PhotoStudio booking service.
func NewService(client *photostudio.Client, enabled bool) *Service {
	return &Service{
		client:  client,
		enabled: enabled,
	}
}

// CreateBooking creates a booking at PhotoStudio.
func (s *Service) CreateBooking(ctx context.Context, userID uuid.UUID, req CreateBookingRequest) (*BookingResponse, error) {
	if !s.enabled {
		return nil, fmt.Errorf("photostudio integration is disabled")
	}

	if s.client == nil {
		return nil, fmt.Errorf("photostudio client is not initialized")
	}

	// Validate time range
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	// Convert to PhotoStudio API request
	psReq := photostudio.BookingRequest{
		RoomID:    req.RoomID,
		StudioID:  req.StudioID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Notes:     req.Notes,
	}

	// Call PhotoStudio API
	resp, err := s.client.CreateBooking(ctx, userID, psReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	if !resp.Success {
		if resp.Error != nil {
			return nil, fmt.Errorf("photostudio error: %s - %s", resp.Error.Code, resp.Error.Message)
		}
		return nil, fmt.Errorf("photostudio returned success=false")
	}

	return &BookingResponse{
		BookingID: resp.Data.Booking.ID,
		Status:    resp.Data.Booking.Status,
	}, nil
}

// GetStudios retrieves studios list from PhotoStudio.
func (s *Service) GetStudios(ctx context.Context, city string, page, limit int) (*StudiosListResponse, error) {
	if !s.enabled {
		return nil, fmt.Errorf("photostudio integration is disabled")
	}

	if s.client == nil {
		return nil, fmt.Errorf("photostudio client is not initialized")
	}

	// Set defaults
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Call PhotoStudio API
	resp, err := s.client.GetStudios(ctx, city, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get studios: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("photostudio returned success=false")
	}

	// Convert response
	studios := make([]StudioResponse, len(resp.Data.Studios))
	for i, studio := range resp.Data.Studios {
		studios[i] = StudioResponse{
			ID:          studio.ID,
			Name:        studio.Name,
			Description: studio.Description,
			Address:     studio.Address,
			City:        studio.City,
			Rating:      studio.Rating,
		}
	}

	result := &StudiosListResponse{
		Studios: studios,
	}
	result.Pagination.Page = resp.Data.Pagination.Page
	result.Pagination.Limit = resp.Data.Pagination.Limit
	result.Pagination.Total = resp.Data.Pagination.Total
	result.Pagination.TotalPages = resp.Data.Pagination.TotalPages

	return result, nil
}

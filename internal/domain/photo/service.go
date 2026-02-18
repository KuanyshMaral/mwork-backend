package photo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/profile"
	uploadDomain "github.com/mwork/mwork-api/internal/domain/upload"
)

// Service handles photo business logic
type Service struct {
	repo          Repository
	modelRepo     profile.ModelRepository
	uploadService *uploadDomain.Service
}

// NewService creates photo service
func NewService(repo Repository, modelRepo profile.ModelRepository, uploadService *uploadDomain.Service) *Service {
	return &Service{
		repo:          repo,
		modelRepo:     modelRepo,
		uploadService: uploadService,
	}
}

// Free tier limit
const FreeTierPhotoLimit = 5

// GeneratePresignedURL creates presigned URL for direct upload
// DEPRECATED: Use /files/init (purpose=portfolio) instead
func (s *Service) GeneratePresignedURL(ctx context.Context, userID uuid.UUID, req *PresignRequest) (*PresignResponse, error) {
	// Get user's model profile
	prof, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil || prof == nil {
		return nil, ErrNoProfileFound
	}

	// Check photo limit (for Free tier)
	count, _ := s.repo.CountByProfile(ctx, prof.ID)
	if count >= FreeTierPhotoLimit {
		// TODO: Check subscription for Pro tier
		return nil, ErrPhotoLimitReached
	}

	// Delegate to new upload service - Stage a multipart upload
	upload, err := s.uploadService.Stage(ctx, userID, uploadDomain.CategoryPhoto, req.Filename, nil)
	if err != nil {
		return nil, err
	}

	return &PresignResponse{
		UploadURL: upload.GetURL(""),
		Key:       upload.ID.String(), // Return upload_id as "key" for backward compatibility
		PublicURL: upload.PermanentURL,
		ExpiresAt: upload.ExpiresAt,
	}, nil
}

// ConfirmUpload registers uploaded file in database
func (s *Service) ConfirmUpload(ctx context.Context, userID uuid.UUID, req *ConfirmUploadRequest) (*Photo, error) {
	// Get user's model profile
	prof, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil || prof == nil {
		return nil, ErrNoProfileFound
	}

	// Check photo limit (for Free tier)
	count, _ := s.repo.CountByProfile(ctx, prof.ID)
	if count >= FreeTierPhotoLimit {
		// TODO: Check subscription for Pro tier
		return nil, ErrPhotoLimitReached
	}

	// Parse upload ID
	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		return nil, ErrUploadNotVerified
	}

	// Get and verify upload from new upload service
	upload, err := s.uploadService.GetByIDForUser(ctx, uploadID, userID)
	if err != nil {
		return nil, ErrUploadNotVerified
	}

	// Ensure upload is committed
	if upload.Status != uploadDomain.StatusCommitted {
		upload, err = s.uploadService.Confirm(ctx, uploadID, userID)
		if err != nil {
			return nil, ErrUploadNotVerified
		}
	}

	// Check if already registered (idempotency) using upload_id
	existing, _ := s.repo.GetByUploadID(ctx, uploadID)
	if existing != nil {
		return existing, nil
	}

	// Get next sort order
	photos, _ := s.repo.ListByProfile(ctx, prof.ID)
	sortOrder := len(photos)

	// Is this the first photo? Make it avatar
	isAvatar := sortOrder == 0

	photo := &Photo{
		ID:           uuid.New(),
		ProfileID:    prof.ID,
		UploadID:     uploadID,
		Key:          upload.PermanentKey,
		URL:          upload.PermanentURL,
		OriginalName: upload.OriginalName,
		MimeType:     upload.MimeType,
		SizeBytes:    upload.Size.Int64,
		IsAvatar:     isAvatar,
		SortOrder:    sortOrder,
		Caption:      req.Caption,
		ProjectName:  req.ProjectName,
		Brand:        req.Brand,
		Year:         req.Year,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, photo); err != nil {
		return nil, err
	}

	return photo, nil
}

// ListByProfile returns all photos for a profile
func (s *Service) ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*Photo, error) {
	return s.repo.ListByProfile(ctx, profileID)
}

// Delete removes a photo
func (s *Service) Delete(ctx context.Context, userID, photoID uuid.UUID) error {
	photo, err := s.repo.GetByID(ctx, photoID)
	if err != nil || photo == nil {
		return ErrPhotoNotFound
	}

	// Check ownership via profile
	prof, _ := s.modelRepo.GetByUserID(ctx, userID)
	if prof == nil || photo.ProfileID != prof.ID {
		return ErrNotPhotoOwner
	}

	// Delete from storage via upload service (if we have upload_id)
	if photo.UploadID != uuid.Nil {
		go s.uploadService.Delete(context.Background(), photo.UploadID, userID)
	}

	// Delete from DB
	if err := s.repo.Delete(ctx, photoID); err != nil {
		return err
	}

	// If was avatar, we need to pick another one?
	// The repo delete logic might handle it or user must set another.
	return nil
}

// SetAvatar sets a photo as profile avatar
func (s *Service) SetAvatar(ctx context.Context, userID, photoID uuid.UUID) (*Photo, error) {
	photo, err := s.repo.GetByID(ctx, photoID)
	if err != nil || photo == nil {
		return nil, ErrPhotoNotFound
	}

	// Check ownership
	prof, _ := s.modelRepo.GetByUserID(ctx, userID)
	if prof == nil || photo.ProfileID != prof.ID {
		return nil, ErrNotPhotoOwner
	}

	// Set avatar in photos table
	if err := s.repo.SetAvatar(ctx, prof.ID, photoID); err != nil {
		return nil, err
	}

	photo.IsAvatar = true
	return photo, nil
}

// ReorderPhotos updates sort order
func (s *Service) ReorderPhotos(ctx context.Context, userID uuid.UUID, photoIDs []uuid.UUID) error {
	prof, _ := s.modelRepo.GetByUserID(ctx, userID)
	if prof == nil {
		return ErrNoProfileFound
	}

	for i, photoID := range photoIDs {
		photo, err := s.repo.GetByID(ctx, photoID)
		if err != nil || photo == nil || photo.ProfileID != prof.ID {
			continue // Skip invalid
		}
		_ = s.repo.UpdateSortOrder(ctx, photoID, i)
	}

	return nil
}

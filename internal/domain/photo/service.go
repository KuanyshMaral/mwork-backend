package photo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/profile"
	"github.com/mwork/mwork-api/internal/pkg/upload"
)

// Service handles photo business logic
type Service struct {
	repo      Repository
	modelRepo profile.ModelRepository
	uploadSvc *upload.Service
}

// NewService creates photo service
func NewService(repo Repository, modelRepo profile.ModelRepository, uploadSvc *upload.Service) *Service {
	return &Service{
		repo:      repo,
		modelRepo: modelRepo,
		uploadSvc: uploadSvc,
	}
}

// Free tier limit
const FreeTierPhotoLimit = 5

// GeneratePresignedURL creates presigned URL for direct upload
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

	// Generate presigned URL
	result, err := s.uploadSvc.GeneratePresignedURL(ctx, userID, req.Filename, req.ContentType, req.Size)
	if err != nil {
		return nil, err
	}

	return &PresignResponse{
		UploadURL: result.UploadURL,
		Key:       result.Key,
		PublicURL: result.PublicURL,
		ExpiresAt: result.ExpiresAt,
	}, nil
}

// ConfirmUpload registers uploaded file in database
func (s *Service) ConfirmUpload(ctx context.Context, userID uuid.UUID, req *ConfirmUploadRequest) (*Photo, error) {
	// Get user's model profile
	prof, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil || prof == nil {
		return nil, ErrNoProfileFound
	}

	// Verify file exists in R2
	metadata, err := s.uploadSvc.VerifyUpload(ctx, req.Key)
	if err != nil {
		return nil, ErrUploadNotVerified
	}

	// Check if already registered (idempotency)
	existing, _ := s.repo.GetByKey(ctx, req.Key)
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
		Key:          req.Key,
		URL:          s.uploadSvc.GetPublicURL(req.Key),
		OriginalName: req.OriginalName,
		MimeType:     metadata.ContentType,
		SizeBytes:    metadata.Size,
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

	// Update profile avatar URL if first photo (assume photos are portfolio items, separate from avatar_url field, but maybe we sync?)
	// Actually logic here updates profile.AvatarURL. But ModelProfile has no AvatarURL field in my previous rewrite?
	// Let's check ModelProfile entity again. It has no AvatarURL. It relies on Photos? or separate field?
	// DB schema `model_profiles` does NOT have `avatar_url`. It seems photos table is the source.
	// Ah no, let me check `viewed_code_item` for model_profiles table.
	// model_profiles table has: id, user_id, name, ... NO avatar_url.
	// So AvatarURL is likely handled by logic.
	// BUT, `profile/entity.go` rewrite REMOVED AvatarURL from ModelProfile struct.
	// So we cannot update it. The avatar is determined by `is_avatar` flag in photos table.
	// Good.

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

	// Delete from R2 (async, don't block)
	go s.uploadSvc.DeleteObject(context.Background(), photo.Key)

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

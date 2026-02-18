-- Add upload_id to photos table to link with uploads table
ALTER TABLE photos ADD COLUMN upload_id UUID;

-- Create index for upload_id lookup
CREATE INDEX idx_photos_upload_id ON photos(upload_id);

-- Add comment
COMMENT ON COLUMN photos.upload_id IS 'Reference to uploads table for file metadata and lifecycle management';

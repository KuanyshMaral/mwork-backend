package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// ProcessedImage contains all variants of a processed image
type ProcessedImage struct {
	Original    []byte
	Thumbnail   []byte
	ContentType string
	Width       int
	Height      int
	ThumbWidth  int
	ThumbHeight int
}

// Config for image processing
type Config struct {
	MaxWidth      int  // Max width for original (default 2000)
	MaxHeight     int  // Max height for original (default 2000)
	ThumbWidth    int  // Thumbnail width (default 300)
	ThumbHeight   int  // Thumbnail height (default 400)
	Quality       int  // JPEG quality 1-100 (default 85)
	ConvertToWebP bool // Convert to WebP format
}

// DefaultConfig returns default processing config
func DefaultConfig() Config {
	return Config{
		MaxWidth:    2000,
		MaxHeight:   2000,
		ThumbWidth:  300,
		ThumbHeight: 400,
		Quality:     85,
	}
}

// Processor handles image processing
type Processor struct {
	config Config
}

// NewProcessor creates image processor
func NewProcessor(config Config) *Processor {
	return &Processor{config: config}
}

// Process processes an image: resize if needed, create thumbnail
func (p *Processor) Process(reader io.Reader, filename string) (*ProcessedImage, error) {
	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	result := &ProcessedImage{
		ContentType: mimeFromFormat(format),
		Width:       img.Bounds().Dx(),
		Height:      img.Bounds().Dy(),
	}

	// Resize if too large
	resized := img
	if result.Width > p.config.MaxWidth || result.Height > p.config.MaxHeight {
		resized = imaging.Fit(img, p.config.MaxWidth, p.config.MaxHeight, imaging.Lanczos)
		result.Width = resized.Bounds().Dx()
		result.Height = resized.Bounds().Dy()
	}

	// Encode original
	original, err := p.encode(resized, format)
	if err != nil {
		return nil, fmt.Errorf("failed to encode original: %w", err)
	}
	result.Original = original

	// Create thumbnail (center crop)
	thumb := imaging.Fill(img, p.config.ThumbWidth, p.config.ThumbHeight, imaging.Center, imaging.Lanczos)
	result.ThumbWidth = thumb.Bounds().Dx()
	result.ThumbHeight = thumb.Bounds().Dy()

	// Encode thumbnail
	thumbnail, err := p.encode(thumb, format)
	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}
	result.Thumbnail = thumbnail

	return result, nil
}

// ValidateType checks if file is a valid image type
func ValidateType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// ValidateSize checks if file size is within limits (in bytes)
func ValidateSize(size int64, maxSize int64) bool {
	return size <= maxSize
}

// MaxFileSize in bytes (10MB)
const MaxFileSize int64 = 10 * 1024 * 1024

// encode encodes image to bytes
func (p *Processor) encode(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: p.config.Quality}); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	default:
		// Default to JPEG for other formats
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: p.config.Quality}); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func mimeFromFormat(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// GeneratePaths generates storage paths for original and thumbnail
func GeneratePaths(userID, filename string) (original, thumb string) {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	original = fmt.Sprintf("photos/%s/%s%s", userID, base, ext)
	thumb = fmt.Sprintf("photos/%s/%s_thumb%s", userID, base, ext)

	return
}

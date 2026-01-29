package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/config"
	"github.com/mwork/mwork-api/internal/pkg/database"
	"github.com/mwork/mwork-api/internal/pkg/storage"
)

const (
	pollInterval    = 5 * time.Second
	maxAttempts     = 3
	maxOriginalSide = 2000
	jpegQuality     = 85
)

type uploadJob struct {
	ID           string `db:"id"`
	PermanentKey string `db:"permanent_key"`
	MimeType     string `db:"mime_type"`
}

func main() {
	cfg := config.Load()
	setupLogger(cfg)

	log.Info().Msg("Starting image-worker")

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer database.ClosePostgres(db)

	rdb, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer database.CloseRedis(rdb)

	r2, err := storage.NewR2Storage(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		AccessKeySecret: cfg.R2AccessKeySecret,
		BucketName:      cfg.R2BucketName,
		PublicURL:       cfg.R2PublicURL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create R2 storage client")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Optional: Redis pub/sub wake-up (polling still runs)
	wake := make(chan struct{}, 1)
	go subscribeWakeups(ctx, rdb, wake)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		log.Info().Msg("Shutdown signal received")
		cancel()
	}()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	lastIdleLog := time.Time{}
	idleLogEvery := 1 * time.Minute

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("image-worker stopped")
			return
		case <-wake:
			// immediate poll
		case <-ticker.C:
		}

		// Process one job at a time (single-threaded MVP)
		job, ok, err := claimNextJob(ctx, db)
		if err != nil {
			log.Error().Err(err).Msg("DB error while claiming job")
			continue
		}
		if !ok {
			now := time.Now()
			if lastIdleLog.IsZero() || now.Sub(lastIdleLog) >= idleLogEvery {
				log.Info().Msg("Idle: no unprocessed uploads found")
				lastIdleLog = now
			}
			continue
		}

		start := time.Now()
		log.Info().
			Str("upload_id", job.ID).
			Str("key", job.PermanentKey).
			Msg("Processing image")

		width, height, err := processOne(ctx, r2, job.PermanentKey)

		if err != nil {
			log.Error().
				Err(err).
				Str("upload_id", job.ID).
				Msg("Processing failed")

			if err2 := markFailed(ctx, db, job.ID, err.Error()); err2 != nil {
				log.Error().Err(err2).Str("upload_id", job.ID).Msg("Failed to update DB status=failed")
			}
			continue
		}

		if err := markDone(ctx, db, job.ID, width, height); err != nil {
			log.Error().Err(err).Str("upload_id", job.ID).Msg("Failed to update DB status=done")
			continue
		}

		log.Info().
			Str("upload_id", job.ID).
			Dur("took", time.Since(start)).
			Int("width", width).
			Int("height", height).
			Msg("Processing done")
	}
}

func processOne(ctx context.Context, st *storage.R2Storage, originalKey string) (int, int, error) {
	// Download
	rc, err := st.Get(ctx, originalKey)
	if err != nil {
		return 0, 0, fmt.Errorf("download: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return 0, 0, fmt.Errorf("read: %w", err)
	}

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, fmt.Errorf("decode: %w", err)
	}

	// Optimize original:
	// - If any side > 2000px => fit into 2000x2000
	// - Save as JPEG quality 85
	opt := img
	if max(imgWidth(img), imgHeight(img)) > maxOriginalSide {
		opt = imaging.Fit(img, maxOriginalSide, maxOriginalSide, imaging.Lanczos)
	}

	var optBuf bytes.Buffer
	if err := imaging.Encode(&optBuf, opt, imaging.JPEG, imaging.JPEGQuality(jpegQuality)); err != nil {
		return 0, 0, fmt.Errorf("encode optimized: %w", err)
	}

	// Overwrite original as JPEG (web-optimized)
	if err := st.Put(ctx, originalKey, bytes.NewReader(optBuf.Bytes()), "image/jpeg"); err != nil {
		return 0, 0, fmt.Errorf("upload optimized: %w", err)
	}

	// Thumbnails
	sizes := []int{200, 400, 800}
	base := strings.TrimSuffix(originalKey, path.Ext(originalKey))

	for _, s := range sizes {
		thumb := imaging.Fit(opt, s, s, imaging.Lanczos)

		var b bytes.Buffer
		if err := imaging.Encode(&b, thumb, imaging.JPEG, imaging.JPEGQuality(jpegQuality)); err != nil {
			return 0, 0, fmt.Errorf("encode thumb %d: %w", s, err)
		}

		thumbKey := fmt.Sprintf("%s_thumb%d.jpg", base, s)
		if err := st.Put(ctx, thumbKey, bytes.NewReader(b.Bytes()), "image/jpeg"); err != nil {
			return 0, 0, fmt.Errorf("upload thumb %d: %w", s, err)
		}
	}

	return imgWidth(opt), imgHeight(opt), nil

}

func claimNextJob(ctx context.Context, db *sqlx.DB) (*uploadJob, bool, error) {
	// Pick candidate
	var j uploadJob
	err := db.GetContext(ctx, &j, `
		SELECT id, permanent_key, mime_type
		FROM uploads
		WHERE status = 'committed'
		  AND permanent_key IS NOT NULL
		  AND permanent_key <> ''
		  AND mime_type IN ('image/jpeg','image/png','image/webp')
		  AND process_status IN ('pending','failed')
		  AND process_attempts < $1
		ORDER BY created_at ASC
		LIMIT 1
	`, maxAttempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	// Claim atomically (safe if multiple workers later)
	res, err := db.ExecContext(ctx, `
		UPDATE uploads
		SET process_status = 'processing',
		    process_attempts = process_attempts + 1,
		    process_error = NULL
		WHERE id = $1
		  AND process_status IN ('pending','failed')
		  AND process_attempts < $2
	`, j.ID, maxAttempts)
	if err != nil {
		return nil, false, err
	}

	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, false, nil
	}

	return &j, true, nil
}

func markDone(ctx context.Context, db *sqlx.DB, id string, width, height int) error {
	_, err := db.ExecContext(ctx, `
		UPDATE uploads
		SET process_status = 'done',
		    processed_at = NOW(),
		    width = $2,
		    height = $3,
		    process_error = NULL
		WHERE id = $1
	`, id, width, height)
	return err
}

func markFailed(ctx context.Context, db *sqlx.DB, id string, msg string) error {
	// keep process_status='failed' (attempts already incremented in claim)
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	_, err := db.ExecContext(ctx, `
		UPDATE uploads
		SET process_status = 'failed',
		    process_error = $2
		WHERE id = $1
	`, id, msg)
	return err
}

func subscribeWakeups(ctx context.Context, rdb *redis.Client, wake chan<- struct{}) {
	// Channel name can be anything; polling is still the main mechanism.
	sub := rdb.Subscribe(ctx, "uploads:confirmed")
	defer func() { _ = sub.Close() }()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.Channel():
			// non-blocking wake-up
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	}
}

func setupLogger(cfg *config.Config) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.IsDevelopment() {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func imgWidth(i image.Image) int {
	return i.Bounds().Dx()
}

func imgHeight(i image.Image) int {
	return i.Bounds().Dy()
}

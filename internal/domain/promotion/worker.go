package promotion

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// Worker handles background tasks for promotions
type Worker struct {
	profileRepo *Repository
	castingRepo *CastingRepository
	interval    time.Duration
	stopCh      chan struct{}
}

// NewWorker creates a new promotion worker
func NewWorker(profileRepo *Repository, castingRepo *CastingRepository, interval time.Duration) *Worker {
	if interval == 0 {
		interval = 1 * time.Hour // Default to running every hour
	}
	return &Worker{
		profileRepo: profileRepo,
		castingRepo: castingRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the background worker
func (w *Worker) Start() {
	log.Info().Msg("Starting promotion worker...")
	go w.loop()
}

// Stop gracefully stops the background worker
func (w *Worker) Stop() {
	log.Info().Msg("Stopping promotion worker...")
	close(w.stopCh)
}

func (w *Worker) loop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run once immediately on startup
	w.processExpirations()

	for {
		select {
		case <-ticker.C:
			w.processExpirations()
		case <-w.stopCh:
			return
		}
	}
}

func (w *Worker) processExpirations() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Debug().Msg("Starting promotion expiration check...")

	// 1. Expire profile promotions
	profileCount, err := w.profileRepo.ExpireCompleted(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to expire completed profile promotions")
	} else if profileCount > 0 {
		log.Info().Int64("count", profileCount).Msg("Expired completed profile promotions")
	}

	// 2. Expire casting promotions
	castingCount, err := w.castingRepo.ExpireCompleted(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to expire completed casting promotions")
	} else if castingCount > 0 {
		log.Info().Int64("count", castingCount).Msg("Expired completed casting promotions")
	}

	log.Debug().Msg("Finished promotion expiration check")
}

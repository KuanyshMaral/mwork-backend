package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// NewRedis creates a new Redis client
// Returns nil if redisURL is empty (Redis is optional for development)
func NewRedis(redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		log.Warn().Msg("Redis URL not configured, running without Redis")
		return nil, nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	// Configure client
	opt.PoolSize = 50
	opt.MinIdleConns = 10
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opt)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	log.Info().Msg("Connected to Redis")
	return client, nil
}

// CloseRedis closes the Redis connection
func CloseRedis(client *redis.Client) {
	if client != nil {
		if err := client.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing Redis connection")
		} else {
			log.Info().Msg("Redis connection closed")
		}
	}
}

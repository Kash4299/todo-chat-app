package redis

import (
	"context"
	"fmt"
	"time"

	"todo/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client wraps the Redis client with additional functionality
type Client struct {
	client *redis.Client
	config *config.RedisConfig
	logger *zap.Logger
}

// NewClient creates a new Redis client
func NewClient(config *config.Config, logger *zap.Logger) (*Client, error) {
	if !config.Redis.Enabled {
		logger.Info("Redis is disabled, skipping connection")
		return &Client{
			client: nil,
			config: &config.Redis,
			logger: logger,
		}, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
		// Connection pool settings
		PoolSize:     10,
		MinIdleConns: 5,
		// Timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		// Retry settings
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis connected successfully",
		zap.String("host", config.Redis.Host),
		zap.String("port", config.Redis.Port),
		zap.Int("db", config.Redis.DB),
	)

	return &Client{
		client: rdb,
		config: &config.Redis,
		logger: logger,
	}, nil
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// IsEnabled returns whether Redis is enabled
func (c *Client) IsEnabled() bool {
	return c.config.Enabled && c.client != nil
}

// Ping tests the Redis connection
func (c *Client) Ping(ctx context.Context) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return c.client.Ping(ctx).Err()
}

// Set sets a key-value pair in Redis
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value from Redis by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("Redis is not enabled")
	}
	return c.client.Get(ctx, key).Result()
}

// Del deletes a key from Redis
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return c.client.Del(ctx, keys...).Err()
}

// Close gracefully closes the Redis connection
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}

	c.logger.Info("Closing Redis connection...")
	if err := c.client.Close(); err != nil {
		c.logger.Error("Error closing Redis connection", zap.Error(err))
		return err
	}
	c.logger.Info("Redis connection closed successfully")
	return nil
}

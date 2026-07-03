package cache

import (
	"context"
	"errors"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}

type Client struct {
	client *redis.Client
}

func New(cfg config.ValkeyConfig) (*Client, error) {
	client := &Client{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
		}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client, client.Ping(ctx)
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("valkey client not initialized")
	}
	return c.client.Ping(ctx).Err()
}

type Checker struct {
	Cache Cache
}

func (c Checker) Name() string {
	return "valkey"
}

func (c Checker) Check(ctx context.Context) error {
	return c.Cache.Ping(ctx)
}

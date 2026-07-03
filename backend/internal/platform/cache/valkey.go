package cache

import (
	"context"
	"errors"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
	valkey "github.com/valkey-io/valkey-go"
)

var ErrNotFound = errors.New("not found")

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}

type Client struct {
	client valkey.Client
}

func New(cfg config.ValkeyConfig) (*Client, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Addr},
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, err
	}
	c := &Client{client: client}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	c.client.Close()
	return nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("valkey client not initialized")
	}
	res := c.client.Do(ctx, c.client.B().Get().Key(key).Build())
	value, err := res.ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return value, nil
}

func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return errors.New("valkey client not initialized")
	}
	if err := c.client.Do(ctx, c.client.B().Set().Key(key).Value(value).Build()).Error(); err != nil {
		return err
	}
	if ttl > 0 {
		seconds := int64(ttl / time.Second)
		if ttl%time.Second != 0 {
			seconds++
		}
		return c.client.Do(ctx, c.client.B().Expire().Key(key).Seconds(seconds).Build()).Error()
	}
	return nil
}

func (c *Client) Del(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return errors.New("valkey client not initialized")
	}
	_, err := c.client.Do(ctx, c.client.B().Del().Key(key).Build()).AsInt64()
	return err
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("valkey client not initialized")
	}
	return c.client.Do(ctx, c.client.B().Ping().Build()).Error()
}

type Checker struct {
	Cache Cache
}

func (c Checker) Name() string { return "valkey" }

func (c Checker) Check(ctx context.Context) error {
	if c.Cache == nil {
		return errors.New("valkey client not initialized")
	}
	return c.Cache.Ping(ctx)
}

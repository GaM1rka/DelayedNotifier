package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"api-service/internal/model"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func New(addr, password string, db int, ttl time.Duration) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db}),
		ttl:    ttl,
	}
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Get(ctx context.Context, id string) (model.Notification, bool, error) {
	data, err := c.client.Get(ctx, key(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return model.Notification{}, false, nil
		}
		return model.Notification{}, false, fmt.Errorf("get cache: %w", err)
	}

	var n model.Notification
	if err := json.Unmarshal(data, &n); err != nil {
		return model.Notification{}, false, fmt.Errorf("decode cache: %w", err)
	}

	return n, true, nil
}

func (c *Cache) Set(ctx context.Context, n model.Notification) error {
	data, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("encode cache: %w", err)
	}
	if err := c.client.Set(ctx, key(n.ID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

func (c *Cache) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, key(id)).Err(); err != nil {
		return fmt.Errorf("delete cache: %w", err)
	}

	return nil
}

func key(id string) string {
	return "notification:" + id
}

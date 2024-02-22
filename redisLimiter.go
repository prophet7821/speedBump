package speedBump

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisCounter struct {
	client       *redis.Client
	windowLength time.Duration
}

// WithRedisLimitCounter FIXME: This function should handle errors gracefully.
func WithRedisLimitCounter(config *Config) Option {
	rc, err := NewRedisLimitCounter(config)
	if err != nil {
		fmt.Println(err)
		return WithNoop()
	}

	return WithLimitCounter(rc)
}

func NewRedisLimitCounter(config *Config) (LimitCounter, error) {
	if config == nil {
		config = &Config{}
	}

	if config.Host == "" {
		config.Host = "127.0.0.1"
	}

	if config.Port < 1 {
		config.Port = 6379
	}

	c, err := newRedisClient(config)
	if err != nil {
		return nil, err
	}

	return &redisCounter{
		client: c,
	}, nil
}

func newRedisClient(config *Config) (*redis.Client, error) {
	if config.Client != nil {
		return config.Client, nil
	}

	var maxIdle, maxActive = config.MaxIdle, config.MaxActive
	if maxIdle < 1 {
		maxIdle = 20
	}

	if maxActive < 1 {
		maxActive = 50
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	c := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     config.Password,
		DB:           config.DBIndex,
		PoolSize:     maxActive,
		MaxIdleConns: maxIdle,
	})

	status := c.Ping(context.Background())
	if status == nil || status.Err() != nil {
		return nil, fmt.Errorf("speedBump : Unable to dial redis %v", status.Err())
	}
	return c, nil
}

func (r *redisCounter) Config(windowLength time.Duration) {
	r.windowLength = windowLength
}

func (r *redisCounter) Inc(key string, currentWindow time.Time) error {
	return r.IncBy(key, currentWindow, 1)
}

func (r *redisCounter) IncBy(key string, currentWindow time.Time, n int) error {
	if n < 1 {
		return nil
	}
	ctx := context.Background()
	conn := r.client
	hkey := limitCounterKey(key, currentWindow)

	if err := conn.IncrBy(ctx, hkey, int64(n)).Err(); err != nil {
		return fmt.Errorf("unable to increment key %s: %v", hkey, err)
	}

	if err := conn.Expire(ctx, hkey, time.Duration(r.windowLength.Seconds()*3)*time.Second).Err(); err != nil {
		return fmt.Errorf("unable to set expiration for key %s: %v", hkey, err)
	}

	return nil
}

func (r *redisCounter) Get(key string, currentWindow time.Time, previousWindow time.Time) (int, int, error) {
	current, err := r.getValue(key, currentWindow)
	if err != nil {
		return 0, 0, err
	}

	previous, err := r.getValue(key, previousWindow)
	if err != nil {
		return 0, 0, err
	}

	return current, previous, nil
}

func (r *redisCounter) getValue(key string, window time.Time) (int, error) {
	val, err := r.client.Get(context.Background(), limitCounterKey(key, window)).Int()
	if errors.Is(err, redis.Nil) {
		return 0, nil

	} else if err != nil {
		return 0, err

	}
	return val, err
}

func limitCounterKey(key string, currentWindow time.Time) string {
	return fmt.Sprintf("speedBump:%d", LimitCounterKey(key, currentWindow))
}

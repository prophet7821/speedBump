package speedBump

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisCounter struct {
	client       *redis.Client
	windowLength time.Duration
}

func WithRedisLimitCounter(config *Config) Option {
	rc, err := NewRedisLimitCounter(config)
	if err != nil {
		panic(err)
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
		return nil, fmt.Errorf("unable to dial redis %v", status.Err())
	}

	return c, nil
}

func (r redisCounter) Config(windowLength time.Duration) {
	r.windowLength = windowLength
}

func (r redisCounter) Inc(key string, currentWindow time.Time) error {
	return r.IncBy(key, currentWindow, 1)
}

func (r redisCounter) IncBy(key string, currentWindow time.Time, n int) error {
	ctx := context.Background()
	conn := r.client
	hkey := LimitCounterKey(key, currentWindow)

	cmd := conn.Do(ctx, "INCRBY", hkey, n)
	if cmd == nil || cmd.Err() != nil {
		return fmt.Errorf("unable to increment key %s: %v", hkey, cmd.Err())
	}

	cmd = conn.Do(ctx, "EXPIRE", hkey, r.windowLength.Seconds()*3)
	if cmd == nil || cmd.Err() != nil {
		return fmt.Errorf("unable to set expiration for key %s: %v", hkey, cmd.Err())
	}

	return nil
}

func (r redisCounter) Get(key string, currentWindow time.Time, previousWindow time.Time) (int, int, error) {
	ctx := context.Background()
	conn := r.client
	hkey := LimitCounterKey(key, currentWindow)

	cmd := conn.Do(ctx, "GET", hkey)
	if cmd == nil || cmd.Err() != nil {
		return 0, 0, fmt.Errorf("unable to get key %s: %v", hkey, cmd.Err())
	}

	current, err := cmd.Int()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse key %s: %v", hkey, err)
	}

	hkey = LimitCounterKey(key, previousWindow)
	cmd = conn.Do(ctx, "GET", hkey)
	if cmd == nil || cmd.Err() != nil {
		return 0, 0, fmt.Errorf("unable to get key %s: %v", hkey, cmd.Err())
	}

	previous, err := cmd.Int()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse key %s: %v", hkey, err)
	}

	return current, previous, nil
}

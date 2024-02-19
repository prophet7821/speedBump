package speedBump

import "github.com/redis/go-redis/v9"

type Config struct {
	Client    *redis.Client
	Host      string
	Port      uint16
	Password  string
	DBIndex   int
	MaxIdle   int
	MaxActive int
	Disabled  bool
}

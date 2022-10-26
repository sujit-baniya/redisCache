package redisCache

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sujit-baniya/framework/contracts/cache"
)

type Config struct {
	Prefix   string
	Host     string
	Port     string
	DB       int
	Password string
}

type Redis struct {
	Prefix string
	Redis  *redis.Client
}

func New(config ...Config) (cache.Store, error) {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == "" {
		cfg.Port = "6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return &Redis{
		Redis:  client,
		Prefix: cfg.Prefix,
	}, nil
}

// Get Retrieve an item from the cache by key.
func (r *Redis) Get(key string, def interface{}) interface{} {
	ctx := context.Background()
	val, err := r.Redis.Get(ctx, r.Prefix+key).Result()
	if err != nil {
		switch s := def.(type) {
		case func() interface{}:
			return s()
		default:
			return def
		}
	}

	return val
}

func (r *Redis) GetBool(key string, def bool) bool {
	res := r.Get(key, def)
	switch res := res.(type) {
	case []byte:
		t := string(res)
		switch t {
		case "1", "true":
			return true
		case "0", "false":
			return false
		}
	case string:
		switch res {
		case "1", "true":
			return true
		case "0", "false":
			return false
		}
	}
	return res.(bool)
}

func (r *Redis) GetInt(key string, def int) int {
	res := r.Get(key, def)
	if val, ok := res.(string); ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return def
		}

		return i
	}

	return res.(int)
}

func (r *Redis) GetString(key string, def string) string {
	return r.Get(key, def).(string)
}

// Has Check an item exists in the cache.
func (r *Redis) Has(key string) bool {
	ctx := context.Background()
	value, err := r.Redis.Exists(ctx, r.Prefix+key).Result()

	if err != nil || value == 0 {
		return false
	}

	return true
}

// Put Store an item in the cache for a given number of seconds.
func (r *Redis) Put(key string, value interface{}, seconds time.Duration) error {
	ctx := context.Background()
	err := r.Redis.Set(ctx, r.Prefix+key, value, seconds).Err()
	if err != nil {
		return err
	}

	return nil
}

// Pull Retrieve an item from the cache and delete it.
func (r *Redis) Pull(key string, def interface{}) interface{} {
	ctx := context.Background()
	val, err := r.Redis.Get(ctx, r.Prefix+key).Result()
	r.Redis.Del(ctx, r.Prefix+key)

	if err != nil {
		return def
	}

	return val
}

// Add Store an item in the cache if the key does not exist.
func (r *Redis) Add(key string, value interface{}, seconds time.Duration) bool {
	ctx := context.Background()
	val, err := r.Redis.SetNX(ctx, r.Prefix+key, value, seconds).Result()
	if err != nil {
		return false
	}

	return val
}

// Remember Get an item from the cache, or execute the given Closure and store the result.
func (r *Redis) Remember(key string, ttl time.Duration, callback func() interface{}) (interface{}, error) {
	val := r.Get(key, nil)

	if val != nil {
		return val, nil
	}

	val = callback()

	if err := r.Put(key, val, ttl); err != nil {
		return nil, err
	}

	return val, nil
}

// RememberForever Get an item from the cache, or execute the given Closure and store the result forever.
func (r *Redis) RememberForever(key string, callback func() interface{}) (interface{}, error) {
	val := r.Get(key, nil)

	if val != nil {
		return val, nil
	}

	val = callback()

	if err := r.Put(key, val, 0); err != nil {
		return nil, err
	}

	return val, nil
}

// Forever Store an item in the cache indefinitely.
func (r *Redis) Forever(key string, value interface{}) bool {
	if err := r.Put(key, value, 0); err != nil {
		return false
	}

	return true
}

// Forget Remove an item from the cache.
func (r *Redis) Forget(key string) bool {
	ctx := context.Background()
	_, err := r.Redis.Del(ctx, r.Prefix+key).Result()

	if err != nil {
		return false
	}

	return true
}

// Flush Remove all items from the cache.
func (r *Redis) Flush() bool {
	ctx := context.Background()
	res, err := r.Redis.FlushAll(ctx).Result()

	if err != nil || res != "OK" {
		return false
	}

	return true
}

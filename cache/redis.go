package cache

import (
	"encoding/json"
	"fmt"
	"time"

	sredis "github.com/hetao29/slightgo/redis"
)

// RedisCache 是基于 Redis 的缓存实现
//
// 对应 SlightPHP 的 SCache 中使用 Memcache/Redis 的模式。
// 复用 slightgo/redis 包的客户端。
type RedisCache struct {
	client *sredis.Client
	prefix string // 键前缀，用于区分不同应用
}

// NewRedis 创建一个新的 Redis 缓存
// prefix 为键前缀，可选
func NewRedis(client *sredis.Client, prefix ...string) *RedisCache {
	c := &RedisCache{
		client: client,
	}
	if len(prefix) > 0 {
		c.prefix = prefix[0]
	}
	return c
}

// key 返回带前缀的完整键名
func (c *RedisCache) key(k string) string {
	if c.prefix != "" {
		return c.prefix + ":" + k
	}
	return k
}

// ---------------------------------------------------------------------------
// Cache 接口实现
// ---------------------------------------------------------------------------

func (c *RedisCache) Get(key string) (interface{}, bool) {
	val, err := c.client.Get(c.key(key))
	if err != nil {
		return nil, false
	}
	return val, true
}

func (c *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	var data string
	switch v := value.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("cache/redis: marshal value: %w", err)
		}
		data = string(b)
	}

	return c.client.Set(c.key(key), data, ttl)
}

func (c *RedisCache) Delete(key string) error {
	_, err := c.client.Del(c.key(key))
	return err
}

func (c *RedisCache) Clear() error {
	// 如果设置了前缀，只清除该前缀下的键
	if c.prefix != "" {
		keys, err := c.client.Client().Keys(c.client.Context(), c.prefix+":*").Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			_, err = c.client.Client().Del(c.client.Context(), keys...).Result()
			return err
		}
		return nil
	}
	_, err := c.client.Client().FlushAll(c.client.Context()).Result()
	return err
}

func (c *RedisCache) Has(key string) bool {
	n, err := c.client.Exists(c.key(key))
	if err != nil {
		return false
	}
	return n > 0
}

func (c *RedisCache) GetMulti(keys []string) map[string]interface{} {
	result := make(map[string]interface{})
	if len(keys) == 0 {
		return result
	}

	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = c.key(k)
	}

	vals, err := c.client.MGet(redisKeys...)
	if err != nil {
		return result
	}

	for i, val := range vals {
		if val != nil {
			result[keys[i]] = val
		}
	}
	return result
}

func (c *RedisCache) SetMulti(items map[string]interface{}, ttl time.Duration) error {
	pipe := c.client.Client().Pipeline()
	for k, v := range items {
		var data string
		switch val := v.(type) {
		case string:
			data = val
		case []byte:
			data = string(val)
		default:
			b, _ := json.Marshal(val)
			data = string(b)
		}
		pipe.Set(c.client.Context(), c.key(k), data, ttl)
	}
	_, err := pipe.Exec(c.client.Context())
	return err
}

func (c *RedisCache) DeleteMulti(keys []string) error {
	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = c.key(k)
	}
	_, err := c.client.Del(redisKeys...)
	return err
}

func (c *RedisCache) Increment(key string, delta int64) (int64, error) {
	return c.client.IncrBy(c.key(key), delta)
}

func (c *RedisCache) Decrement(key string, delta int64) (int64, error) {
	return c.client.DecrBy(c.key(key), delta)
}

package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCache 是基于内存的缓存实现
//
// 对应 SlightPHP SCache 的 File/APC 缓存模式。
// 使用 sync.RWMutex 保证并发安全，
// 定期清理过期键值。
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*Item
	closeCh  chan struct{}
	interval time.Duration
	stats    struct {
		hits   int64
		misses int64
	}
}

// NewMemory 创建一个新的内存缓存
// interval 指定清理过期键值的时间间隔（默认 1 分钟）
func NewMemory(cleanupInterval ...time.Duration) *MemoryCache {
	interval := 1 * time.Minute
	if len(cleanupInterval) > 0 && cleanupInterval[0] > 0 {
		interval = cleanupInterval[0]
	}

	c := &MemoryCache{
		items:    make(map[string]*Item),
		closeCh:  make(chan struct{}),
		interval: interval,
	}

	// 启动定期清理
	go c.cleanupLoop()

	return c
}

// cleanupLoop 定期清理过期缓存
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.closeCh:
			return
		}
	}
}

// deleteExpired 删除所有过期的缓存项
func (c *MemoryCache) deleteExpired() {
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, k)
		}
	}
}

// ---------------------------------------------------------------------------
// Cache 接口实现
// ---------------------------------------------------------------------------

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		atomic.AddInt64(&c.stats.misses, 1)
		return nil, false
	}

	if item.IsExpired() {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		atomic.AddInt64(&c.stats.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&c.stats.hits, 1)
	return item.Value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	c.items[key] = &Item{
		Key:        key,
		Value:      value,
		Expiration: exp,
	}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	c.items = make(map[string]*Item)
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Has(key string) bool {
	_, ok := c.Get(key)
	return ok
}

func (c *MemoryCache) GetMulti(keys []string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if v, ok := c.Get(key); ok {
			result[key] = v
		}
	}
	return result
}

func (c *MemoryCache) SetMulti(items map[string]interface{}, ttl time.Duration) error {
	for k, v := range items {
		if err := c.Set(k, v, ttl); err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryCache) DeleteMulti(keys []string) error {
	for _, key := range keys {
		if err := c.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryCache) Increment(key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		c.items[key] = &Item{
			Key:   key,
			Value: delta,
		}
		return delta, nil
	}

	switch v := item.Value.(type) {
	case int:
		item.Value = v + int(delta)
		return int64(v + int(delta)), nil
	case int64:
		item.Value = v + delta
		return v + delta, nil
	case float64:
		item.Value = v + float64(delta)
		return int64(v + float64(delta)), nil
	default:
		return 0, nil
	}
}

func (c *MemoryCache) Decrement(key string, delta int64) (int64, error) {
	return c.Increment(key, -delta)
}

// ---------------------------------------------------------------------------
// 扩展方法
// ---------------------------------------------------------------------------

// Len 返回缓存项数量
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stats 返回缓存命中/未命中统计
func (c *MemoryCache) Stats() (hits, misses int64) {
	return atomic.LoadInt64(&c.stats.hits), atomic.LoadInt64(&c.stats.misses)
}

// Close 关闭缓存，停止清理协程
func (c *MemoryCache) Close() {
	close(c.closeCh)
}

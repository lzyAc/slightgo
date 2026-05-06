// Package cache 提供缓存接口和多种实现
//
// 对应 SlightPHP 的 SCache 插件。
// 支持 Memory、Redis、File 三种缓存后端。
package cache

import "time"

// ---------------------------------------------------------------------------
// Cache 接口
// ---------------------------------------------------------------------------

// Cache 定义缓存存储接口
type Cache interface {
	// Get 获取缓存值
	Get(key string) (interface{}, bool)

	// Set 设置缓存
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	Delete(key string) error

	// Clear 清空所有缓存
	Clear() error

	// Has 检查键是否存在
	Has(key string) bool

	// GetMulti 批量获取缓存值
	GetMulti(keys []string) map[string]interface{}

	// SetMulti 批量设置缓存
	SetMulti(items map[string]interface{}, ttl time.Duration) error

	// DeleteMulti 批量删除缓存
	DeleteMulti(keys []string) error

	// Increment 自增
	Increment(key string, delta int64) (int64, error)

	// Decrement 自减
	Decrement(key string, delta int64) (int64, error)
}

// ---------------------------------------------------------------------------
// 公共类型
// ---------------------------------------------------------------------------

// Item 表示一个缓存项
type Item struct {
	Key        string
	Value      interface{}
	Expiration int64 // Unix 时间戳，0 表示永不过期
}

// IsExpired 检查缓存项是否已过期
func (item *Item) IsExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// ---------------------------------------------------------------------------
// 默认缓存实例
// ---------------------------------------------------------------------------

var defaultCache Cache

// SetDefaultCache 设置默认缓存实例
func SetDefaultCache(c Cache) {
	defaultCache = c
}

// DefaultCache 获取默认缓存实例
func DefaultCache() Cache {
	return defaultCache
}

// Get 使用默认缓存获取值
func Get(key string) (interface{}, bool) {
	if defaultCache == nil {
		return nil, false
	}
	return defaultCache.Get(key)
}

// Set 使用默认缓存设置值
func Set(key string, value interface{}, ttl time.Duration) error {
	if defaultCache == nil {
		return nil
	}
	return defaultCache.Set(key, value, ttl)
}

// Delete 使用默认缓存删除值
func Delete(key string) error {
	if defaultCache == nil {
		return nil
	}
	return defaultCache.Delete(key)
}

// Package redis 提供 Redis 客户端封装
//
// 对应 SlightPHP 的 SRedis 插件。
// 基于 go-redis/redis 库，支持连接池配置、
// 错误连接自动重置，更好地支持长连接场景。
package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// Redis 客户端
// ---------------------------------------------------------------------------

// Client 封装了 go-redis 客户端，提供便捷操作方法
type Client struct {
	client  *goredis.Client
	ctx     context.Context
	options *Options
}

// Options Redis 连接选项
type Options struct {
	// Addr 服务器地址 (host:port)
	Addr string

	// Password 密码
	Password string

	// DB 数据库编号
	DB int

	// PoolSize 连接池大小
	PoolSize int

	// MinIdleConns 最小空闲连接数
	MinIdleConns int

	// DialTimeout 连接超时
	DialTimeout time.Duration

	// ReadTimeout 读取超时
	ReadTimeout time.Duration

	// WriteTimeout 写入超时
	WriteTimeout time.Duration

	// PoolTimeout 获取连接超时
	PoolTimeout time.Duration

	// MaxRetries 最大重试次数
	MaxRetries int
}

// DefaultOptions 返回默认选项
func DefaultOptions() *Options {
	return &Options{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		MaxRetries:   3,
	}
}

// New 创建一个新的 Redis 客户端
func New(opts *Options) *Client {
	if opts == nil {
		opts = DefaultOptions()
	}

	c := &Client{
		ctx:     context.Background(),
		options: opts,
	}

	c.client = goredis.NewClient(&goredis.Options{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		PoolSize:     opts.PoolSize,
		MinIdleConns: opts.MinIdleConns,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolTimeout:  opts.PoolTimeout,
		MaxRetries:   opts.MaxRetries,
	})

	return c
}

// NewFromURL 从 URL 创建 Redis 客户端
// redis://[[user]:[password]]@host:port[/db]
func NewFromURL(redisURL string) (*Client, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: goredis.NewClient(opts),
		ctx:    context.Background(),
	}, nil
}

// ---------------------------------------------------------------------------
// 连接管理
// ---------------------------------------------------------------------------

// Ping 测试连接
func (c *Client) Ping() error {
	return c.client.Ping(c.ctx).Err()
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}

// Client 返回原始 go-redis 客户端
func (c *Client) Client() *goredis.Client {
	return c.client
}

// Context 返回当前上下文
func (c *Client) Context() context.Context {
	return c.ctx
}

// SetContext 设置上下文
func (c *Client) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// ---------------------------------------------------------------------------
// String 操作
// ---------------------------------------------------------------------------

// Set 设置键值
func (c *Client) Set(key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(c.ctx, key, value, ttl).Err()
}

// Get 获取键值
func (c *Client) Get(key string) (string, error) {
	return c.client.Get(c.ctx, key).Result()
}

// GetSet 设置新值并返回旧值
func (c *Client) GetSet(key string, value interface{}) (string, error) {
	return c.client.GetSet(c.ctx, key, value).Result()
}

// SetNX 仅在键不存在时设置值
func (c *Client) SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	return c.client.SetNX(c.ctx, key, value, ttl).Result()
}

// MSet 批量设置
func (c *Client) MSet(values ...interface{}) error {
	return c.client.MSet(c.ctx, values...).Err()
}

// MGet 批量获取
func (c *Client) MGet(keys ...string) ([]interface{}, error) {
	return c.client.MGet(c.ctx, keys...).Result()
}

// Del 删除键
func (c *Client) Del(keys ...string) (int64, error) {
	return c.client.Del(c.ctx, keys...).Result()
}

// Exists 检查键是否存在
func (c *Client) Exists(keys ...string) (int64, error) {
	return c.client.Exists(c.ctx, keys...).Result()
}

// Expire 设置过期时间
func (c *Client) Expire(key string, ttl time.Duration) (bool, error) {
	return c.client.Expire(c.ctx, key, ttl).Result()
}

// TTL 获取剩余过期时间
func (c *Client) TTL(key string) (time.Duration, error) {
	return c.client.TTL(c.ctx, key).Result()
}

// Incr 自增
func (c *Client) Incr(key string) (int64, error) {
	return c.client.Incr(c.ctx, key).Result()
}

// IncrBy 按指定步长自增
func (c *Client) IncrBy(key string, delta int64) (int64, error) {
	return c.client.IncrBy(c.ctx, key, delta).Result()
}

// Decr 自减
func (c *Client) Decr(key string) (int64, error) {
	return c.client.Decr(c.ctx, key).Result()
}

// DecrBy 按指定步长自减
func (c *Client) DecrBy(key string, delta int64) (int64, error) {
	return c.client.DecrBy(c.ctx, key, delta).Result()
}

// ---------------------------------------------------------------------------
// Hash 操作
// ---------------------------------------------------------------------------

// HSet 设置哈希字段
func (c *Client) HSet(key string, values ...interface{}) (int64, error) {
	return c.client.HSet(c.ctx, key, values...).Result()
}

// HGet 获取哈希字段
func (c *Client) HGet(key, field string) (string, error) {
	return c.client.HGet(c.ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func (c *Client) HGetAll(key string) (map[string]string, error) {
	return c.client.HGetAll(c.ctx, key).Result()
}

// HDel 删除哈希字段
func (c *Client) HDel(key string, fields ...string) (int64, error) {
	return c.client.HDel(c.ctx, key, fields...).Result()
}

// HExists 检查哈希字段是否存在
func (c *Client) HExists(key, field string) (bool, error) {
	return c.client.HExists(c.ctx, key, field).Result()
}

// HKeys 获取所有哈希字段名
func (c *Client) HKeys(key string) ([]string, error) {
	return c.client.HKeys(c.ctx, key).Result()
}

// HVals 获取所有哈希字段值
func (c *Client) HVals(key string) ([]string, error) {
	return c.client.HVals(c.ctx, key).Result()
}

// HLen 获取哈希字段数量
func (c *Client) HLen(key string) (int64, error) {
	return c.client.HLen(c.ctx, key).Result()
}

// ---------------------------------------------------------------------------
// List 操作
// ---------------------------------------------------------------------------

// LPush 从左侧推入列表
func (c *Client) LPush(key string, values ...interface{}) (int64, error) {
	return c.client.LPush(c.ctx, key, values...).Result()
}

// RPush 从右侧推入列表
func (c *Client) RPush(key string, values ...interface{}) (int64, error) {
	return c.client.RPush(c.ctx, key, values...).Result()
}

// LPop 从左侧弹出列表
func (c *Client) LPop(key string) (string, error) {
	return c.client.LPop(c.ctx, key).Result()
}

// RPop 从右侧弹出列表
func (c *Client) RPop(key string) (string, error) {
	return c.client.RPop(c.ctx, key).Result()
}

// LLen 获取列表长度
func (c *Client) LLen(key string) (int64, error) {
	return c.client.LLen(c.ctx, key).Result()
}

// LRange 获取列表范围
func (c *Client) LRange(key string, start, stop int64) ([]string, error) {
	return c.client.LRange(c.ctx, key, start, stop).Result()
}

// ---------------------------------------------------------------------------
// Set 操作
// ---------------------------------------------------------------------------

// SAdd 向集合添加元素
func (c *Client) SAdd(key string, members ...interface{}) (int64, error) {
	return c.client.SAdd(c.ctx, key, members...).Result()
}

// SMembers 获取集合所有元素
func (c *Client) SMembers(key string) ([]string, error) {
	return c.client.SMembers(c.ctx, key).Result()
}

// SIsMember 检查元素是否在集合中
func (c *Client) SIsMember(key string, member interface{}) (bool, error) {
	return c.client.SIsMember(c.ctx, key, member).Result()
}

// SRem 从集合中移除元素
func (c *Client) SRem(key string, members ...interface{}) (int64, error) {
	return c.client.SRem(c.ctx, key, members...).Result()
}

// SCard 获取集合基数
func (c *Client) SCard(key string) (int64, error) {
	return c.client.SCard(c.ctx, key).Result()
}

// ---------------------------------------------------------------------------
// SortedSet 操作
// ---------------------------------------------------------------------------

// ZAdd 向有序集合添加元素
func (c *Client) ZAdd(key string, members ...goredis.Z) (int64, error) {
	return c.client.ZAdd(c.ctx, key, members...).Result()
}

// ZRange 获取有序集合范围
func (c *Client) ZRange(key string, start, stop int64) ([]string, error) {
	return c.client.ZRange(c.ctx, key, start, stop).Result()
}

// ZRevRange 反向获取有序集合范围
func (c *Client) ZRevRange(key string, start, stop int64) ([]string, error) {
	return c.client.ZRevRange(c.ctx, key, start, stop).Result()
}

// ZRem 从有序集合中移除元素
func (c *Client) ZRem(key string, members ...interface{}) (int64, error) {
	return c.client.ZRem(c.ctx, key, members...).Result()
}

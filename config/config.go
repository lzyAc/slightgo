// Package config 提供配置加载功能
//
// 对应 SlightPHP 的 SConfig 插件。
// 支持 JSON 格式的配置文件，按 Zone 环境加载。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config 配置管理器
type Config struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	loaded bool
}

// New 创建一个新的配置管理器
func New() *Config {
	return &Config{
		data: make(map[string]interface{}),
	}
}

// Load 从 JSON 文件加载配置
// filePath 可以是绝对路径或相对路径
func (c *Config) Load(filePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("config: cannot resolve path %s: %w", filePath, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("config: cannot read file %s: %w", absPath, err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("config: cannot parse JSON from %s: %w", absPath, err)
	}

	c.data = parsed
	c.loaded = true
	return nil
}

// LoadString 从 JSON 字符串加载配置
func (c *Config) LoadString(jsonStr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return fmt.Errorf("config: cannot parse JSON string: %w", err)
	}

	c.data = parsed
	c.loaded = true
	return nil
}

// Get 获取配置值，支持通过点号分隔的路径访问嵌套字段
// 例如: Get("database.host") 返回 data["database"]["host"]
func (c *Config) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.loaded {
		return nil
	}

	return getNested(c.data, key)
}

// GetString 获取字符串配置值
func (c *Config) GetString(key string) string {
	v := c.Get(key)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetInt 获取整数配置值（支持 float64 JSON 自动转换）
func (c *Config) GetInt(key string) int {
	v := c.Get(key)
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// GetBool 获取布尔配置值
func (c *Config) GetBool(key string) bool {
	v := c.Get(key)
	if v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}

// GetMap 获取 map[string]interface{} 配置值
func (c *Config) GetMap(key string) map[string]interface{} {
	v := c.Get(key)
	if v == nil {
		return nil
	}
	m, _ := v.(map[string]interface{})
	return m
}

// GetSlice 获取 []interface{} 配置值
func (c *Config) GetSlice(key string) []interface{} {
	v := c.Get(key)
	if v == nil {
		return nil
	}
	s, _ := v.([]interface{})
	return s
}

// All 返回所有配置数据
func (c *Config) All() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// Set 设置配置值（运行时修改）
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.loaded {
		c.data = make(map[string]interface{})
		c.loaded = true
	}

	setNested(c.data, key, value)
}

// Reset 重置所有配置
func (c *Config) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]interface{})
	c.loaded = false
}

// IsLoaded 返回配置是否已加载
func (c *Config) IsLoaded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loaded
}

// ---------------------------------------------------------------------------
// 嵌套访问辅助
// ---------------------------------------------------------------------------

func getNested(data map[string]interface{}, key string) interface{} {
	if data == nil {
		return nil
	}

	// 尝试直接获取
	if v, ok := data[key]; ok {
		return v
	}

	// 尝试点号分隔路径
	parts := splitKey(key)
	if len(parts) == 0 {
		return nil
	}

	current := interface{}(data)
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

func setNested(data map[string]interface{}, key string, value interface{}) {
	parts := splitKey(key)
	if len(parts) == 0 {
		return
	}

	if len(parts) == 1 {
		data[parts[0]] = value
		return
	}

	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			next := make(map[string]interface{})
			current[part] = next
			current = next
		}
	}
}

func splitKey(key string) []string {
	if key == "" {
		return nil
	}

	var parts []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			if i > start {
				parts = append(parts, key[start:i])
			}
			start = i + 1
		}
	}
	if start < len(key) {
		parts = append(parts, key[start:])
	}
	return parts
}

// ---------------------------------------------------------------------------
// 包级别便捷函数
// ---------------------------------------------------------------------------

var defaultConfig = New()

// Load 加载默认配置
func Load(filePath string) error {
	return defaultConfig.Load(filePath)
}

// Get 获取默认配置
func Get(key string) interface{} {
	return defaultConfig.Get(key)
}

// GetString 获取默认字符串配置
func GetString(key string) string {
	return defaultConfig.GetString(key)
}

// GetInt 获取默认整数配置
func GetInt(key string) int {
	return defaultConfig.GetInt(key)
}

// SetConfig 设置默认配置实例（用于自定义配置管理器）
func SetConfig(c *Config) {
	defaultConfig = c
}

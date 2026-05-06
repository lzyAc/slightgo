package slightgo

import (
	"github.com/hetao29/slightgo/config"
)

// Config 是对 config.Config 的封装，方便从 slightgo 包直接访问
type Config struct {
	*config.Config
}

// NewConfig 创建一个新的配置实例
func (s *SlightGo) NewConfig() *Config {
	return &Config{Config: config.New()}
}

// LoadConfig 从 JSON 文件加载配置
func (s *SlightGo) LoadConfig(filePath string) (*Config, error) {
	cfg := config.New()
	if err := cfg.Load(filePath); err != nil {
		return nil, err
	}
	s.config = &Config{Config: cfg}
	return s.config, nil
}

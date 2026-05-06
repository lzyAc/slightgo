// Package tpl 提供模板引擎功能
//
// 对应 SlightPHP 的 STpl 插件。
// 基于 Go 标准库 html/template，支持模板缓存、自定义函数等特性。
package tpl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// 模板引擎
// ---------------------------------------------------------------------------

// Engine 是模板渲染引擎
type Engine struct {
	mu         sync.RWMutex
	dir        string            // 模板文件目录
	ext        string            // 模板文件扩展名
	cache      map[string]*template.Template
	funcMap    template.FuncMap
	autoReload bool
	leftDelim  string
	rightDelim string
	fileSystem fs.FS // 可选的文件系统
}

// Option 配置选项
type Option func(*Engine)

// WithDir 设置模板目录
func WithDir(dir string) Option {
	return func(e *Engine) {
		e.dir = dir
	}
}

// WithExt 设置模板文件扩展名（默认 .html）
func WithExt(ext string) Option {
	return func(e *Engine) {
		e.ext = ext
	}
}

// WithAutoReload 设置是否自动重载模板（开发模式）
func WithAutoReload(reload bool) Option {
	return func(e *Engine) {
		e.autoReload = reload
	}
}

// WithDelims 设置模板分隔符
func WithDelims(left, right string) Option {
	return func(e *Engine) {
		e.leftDelim = left
		e.rightDelim = right
	}
}

// WithFuncMap 设置模板函数
func WithFuncMap(fm template.FuncMap) Option {
	return func(e *Engine) {
		for k, v := range fm {
			e.funcMap[k] = v
		}
	}
}

// WithFileSystem 设置自定义文件系统
func WithFileSystem(fsys fs.FS) Option {
	return func(e *Engine) {
		e.fileSystem = fsys
	}
}

// New 创建一个新的模板引擎
func New(opts ...Option) *Engine {
	e := &Engine{
		dir:        "templates",
		ext:        ".html",
		cache:      make(map[string]*template.Template),
		funcMap:    make(template.FuncMap),
		autoReload: false,
		leftDelim:  "{{",
		rightDelim: "}}",
	}

	// 添加默认模板函数
	for k, v := range DefaultFuncMap() {
		e.funcMap[k] = v
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ---------------------------------------------------------------------------
// 模板函数
// ---------------------------------------------------------------------------

// AddFunc 添加模板函数
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.funcMap[name] = fn
}

// ---------------------------------------------------------------------------
// 模板渲染 (字符串)
// ---------------------------------------------------------------------------

// Render 渲染模板字符串
func (e *Engine) Render(name, tmpl string, data interface{}) (string, error) {
	t := template.New(name).Delims(e.leftDelim, e.rightDelim).Funcs(e.funcMap)
	t, err := t.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("tpl: parse template '%s': %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("tpl: execute template '%s': %w", name, err)
	}

	return buf.String(), nil
}

// RenderToWriter 渲染模板到 writer
func (e *Engine) RenderToWriter(name, tmpl string, data interface{}, w io.Writer) error {
	t := template.New(name).Delims(e.leftDelim, e.rightDelim).Funcs(e.funcMap)
	t, err := t.Parse(tmpl)
	if err != nil {
		return fmt.Errorf("tpl: parse template '%s': %w", name, err)
	}
	return t.Execute(w, data)
}

// ---------------------------------------------------------------------------
// 模板文件渲染
// ---------------------------------------------------------------------------

// Fetch 获取并渲染模板文件
// name 是模板文件名（不含扩展名）
func (e *Engine) Fetch(name string, data interface{}) (string, error) {
	t, err := e.getTemplate(name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("tpl: execute '%s': %w", name, err)
	}
	return buf.String(), nil
}

// Display 渲染模板并写入 ResponseWriter
func (e *Engine) Display(w io.Writer, name string, data interface{}) error {
	t, err := e.getTemplate(name)
	if err != nil {
		return err
	}
	if err := t.Execute(w, data); err != nil {
		return fmt.Errorf("tpl: display '%s': %w", name, err)
	}
	return nil
}

// getTemplate 获取模板，支持缓存
func (e *Engine) getTemplate(name string) (*template.Template, error) {
	// 自动重载模式：每次都重新解析
	if e.autoReload {
		return e.loadTemplate(name)
	}

	// 缓存模式
	e.mu.RLock()
	t, ok := e.cache[name]
	e.mu.RUnlock()

	if ok {
		return t, nil
	}

	// 缓存未命中，加载并缓存
	t, err := e.loadTemplate(name)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[name] = t
	e.mu.Unlock()

	return t, nil
}

// loadTemplate 从文件加载模板
func (e *Engine) loadTemplate(name string) (*template.Template, error) {
	tmplPath := filepath.Join(e.dir, name+e.ext)
	tmplName := name + e.ext

	t := template.New(tmplName).Delims(e.leftDelim, e.rightDelim).Funcs(e.funcMap)

	var content string
	if e.fileSystem != nil {
		data, err := fs.ReadFile(e.fileSystem, tmplPath)
		if err != nil {
			return nil, fmt.Errorf("tpl: read template '%s': %w", tmplPath, err)
		}
		content = string(data)
	} else {
		data, err := os.ReadFile(tmplPath)
		if err != nil {
			return nil, fmt.Errorf("tpl: read template file '%s': %w", tmplPath, err)
		}
		content = string(data)
	}

	// 解析主模板
	t, err := t.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("tpl: parse template '%s': %w", tmplPath, err)
	}

	// 自动解析目录中所有模板文件，支持模板组合
	e.parseSubTemplates(t)

	return t, nil
}

// parseSubTemplates 解析目录中的子模板文件
func (e *Engine) parseSubTemplates(main *template.Template) {
	if e.fileSystem != nil {
		entries, err := fs.ReadDir(e.fileSystem, e.dir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), e.ext) {
					data, _ := fs.ReadFile(e.fileSystem, filepath.Join(e.dir, entry.Name()))
					main.Parse(string(data))
				}
			}
		}
	} else {
		filepath.Walk(e.dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), e.ext) {
				data, _ := os.ReadFile(path)
				main.Parse(string(data))
			}
			return nil
		})
	}
}

// ---------------------------------------------------------------------------
// 缓存管理
// ---------------------------------------------------------------------------

// ClearCache 清空模板缓存
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]*template.Template)
}

// ---------------------------------------------------------------------------
// 便捷函数
// ---------------------------------------------------------------------------

var defaultEngine = New()

// SetDefaultEngine 设置全局默认模板引擎
func SetDefaultEngine(e *Engine) {
	defaultEngine = e
}

// Fetch 使用默认引擎渲染模板
func Fetch(name string, data interface{}) (string, error) {
	return defaultEngine.Fetch(name, data)
}

// Render 使用默认引擎渲染字符串模板
func Render(name, tmpl string, data interface{}) (string, error) {
	return defaultEngine.Render(name, tmpl, data)
}

// DefaultEngine 返回默认模板引擎
func DefaultEngine() *Engine {
	return defaultEngine
}

// ---------------------------------------------------------------------------
// 内置模板函数
// ---------------------------------------------------------------------------

// DefaultFuncMap 返回默认的模板函数映射
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
		"join":  strings.Join,
		"split": strings.Split,
		"len": func(items interface{}) int {
			switch v := items.(type) {
			case []interface{}:
				return len(v)
			case string:
				return len(v)
			default:
				return 0
			}
		},
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"safe": func(html string) template.HTML {
			return template.HTML(html)
		},
	}
}

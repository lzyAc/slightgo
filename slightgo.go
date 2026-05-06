// Package slightgo — SlightPHP 的 Go 语言版本
//
// SlightGo 是一个轻量级、高效的 Go Web 开发框架，
// 遵循 SlightPHP 的 Zone → Page → Entry 三层路由架构。
package slightgo

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
)

// ---------------------------------------------------------------------------
// SlightGo 核心结构
// ---------------------------------------------------------------------------

// SlightGo 是框架的核心实例，管理路由、配置和HTTP服务器。
type SlightGo struct {
	defaultZone  string
	defaultPage  string
	defaultEntry string
	appDir       string
	splitFlag    string
	routes       map[string]map[string]interface{} // zone -> page -> controller
	middlewares  []MiddlewareFunc
	config       *Config
	server       *http.Server
}

// MiddlewareFunc 定义中间件函数签名
type MiddlewareFunc func(next http.Handler) http.Handler

// ---------------------------------------------------------------------------
// 选项函数
// ---------------------------------------------------------------------------

// Option 定义框架配置选项
type Option func(*SlightGo)

// WithDefaultZone 设置默认 Zone
func WithDefaultZone(zone string) Option {
	return func(s *SlightGo) {
		s.defaultZone = zone
	}
}

// WithDefaultPage 设置默认 Page
func WithDefaultPage(page string) Option {
	return func(s *SlightGo) {
		s.defaultPage = page
	}
}

// WithDefaultEntry 设置默认 Entry
func WithDefaultEntry(entry string) Option {
	return func(s *SlightGo) {
		s.defaultEntry = entry
	}
}

// WithAppDir 设置应用目录
func WithAppDir(dir string) Option {
	return func(s *SlightGo) {
		s.appDir = dir
	}
}

// WithSplitFlag 设置URL分隔符
func WithSplitFlag(flag string) Option {
	return func(s *SlightGo) {
		s.splitFlag = flag
	}
}

// ---------------------------------------------------------------------------
// 构造函数
// ---------------------------------------------------------------------------

// New 创建一个新的 SlightGo 实例
func New(opts ...Option) *SlightGo {
	s := &SlightGo{
		defaultZone:  "zone",
		defaultPage:  "page",
		defaultEntry: "entry",
		appDir:       ".",
		splitFlag:    "/",
		routes:       make(map[string]map[string]interface{}),
		middlewares:  make([]MiddlewareFunc, 0),
		config:      &Config{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ---------------------------------------------------------------------------
// 路由注册
// ---------------------------------------------------------------------------

// Zone 注册一个 Zone（模块），在回调中注册 Pages
func (s *SlightGo) Zone(name string, fn func(z *Zone)) *SlightGo {
	z := &Zone{
		name:    name,
		pages:   make(map[string]interface{}),
		app:     s,
	}
	fn(z)
	s.routes[name] = z.pages
	return s
}

// Zone 代表一个模块，对应 URL 路径的第一段
type Zone struct {
	name  string
	pages map[string]interface{}
	app   *SlightGo
}

// Page 在 Zone 中注册一个 Page（控制器）
func (z *Zone) Page(name string, controller interface{}) *Zone {
	z.pages[name] = controller
	return z
}

// Register 直接注册一个控制器到指定 Zone 和 Page
// controller 必须实现 Entry 接口（包含 Page* 方法）
func (s *SlightGo) Register(zone, page string, controller interface{}) *SlightGo {
	if _, ok := s.routes[zone]; !ok {
		s.routes[zone] = make(map[string]interface{})
	}
	s.routes[zone][page] = controller
	return s
}

// ---------------------------------------------------------------------------
// 中间件
// ---------------------------------------------------------------------------

// Use 注册全局中间件
func (s *SlightGo) Use(mw MiddlewareFunc) *SlightGo {
	s.middlewares = append(s.middlewares, mw)
	return s
}

// ---------------------------------------------------------------------------
// HTTP 服务器
// ---------------------------------------------------------------------------

// Run 启动 HTTP 服务器
func (s *SlightGo) Run(addr ...string) *SlightGo {
	address := ":8080"
	if len(addr) > 0 {
		address = addr[0]
	}

	var handler http.Handler = http.HandlerFunc(s.dispatch)

	// 从后往前包装中间件
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		handler = s.middlewares[i](handler)
	}

	s.server = &http.Server{
		Addr:    address,
		Handler: handler,
	}

	fmt.Fprintf(os.Stdout, "[SlightGo] server running on %s\n", address)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[SlightGo] server error: %v", err)
	}
	return s
}

// ---------------------------------------------------------------------------
// URL 调度器 (Zone → Page → Entry)
// ---------------------------------------------------------------------------

func (s *SlightGo) dispatch(w http.ResponseWriter, r *http.Request) {
	// 解析 path 信息
	pathInfo := s.parsePath(r.URL.Path)

	// 解析 Zone, Page, Entry
	zoneName := s.defaultZone
	pageName := s.defaultPage
	entryName := s.defaultEntry

	if len(pathInfo) > 0 && pathInfo[0] != "" {
		zoneName = pathInfo[0]
	}
	if len(pathInfo) > 1 && pathInfo[1] != "" {
		pageName = pathInfo[1]
	}
	if len(pathInfo) > 2 && pathInfo[2] != "" {
		entryName = pathInfo[2]
	}

	// 构建 inPath (包含所有分段)
	inPath := pathInfo

	// 查找路由
	zoneRoutes, zoneOK := s.routes[zoneName]
	if !zoneOK {
		s.handleNotFound(w, r, zoneName, pageName, entryName)
		return
	}

	controller, pageOK := zoneRoutes[pageName]
	if !pageOK {
		s.handleNotFound(w, r, zoneName, pageName, entryName)
		return
	}

	// 查找并调用 Entry 方法
	if err := s.callEntry(w, r, controller, entryName, inPath); err != nil {
		s.handleError(w, r, err)
	}
}

func (s *SlightGo) parsePath(urlPath string) []string {
	// 去除 query string
	if idx := strings.IndexByte(urlPath, '?'); idx != -1 {
		urlPath = urlPath[:idx]
	}

	// 按分隔符分割
	parts := strings.Split(strings.Trim(urlPath, s.splitFlag), s.splitFlag)

	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (s *SlightGo) callEntry(w http.ResponseWriter, r *http.Request, controller interface{}, entryName string, inPath []string) error {
	v := reflect.ValueOf(controller)
	// Entry 方法名: Page + 首字母大写的 Entry 名
	methodName := "Page" + toTitle(entryName)
	method := v.MethodByName(methodName)

	if !method.IsValid() {
		return fmt.Errorf("slightgo: entry '%s' not found (expected method %s)", entryName, methodName)
	}

	// 准备参数
	methodType := method.Type()
	var args []reflect.Value

	if methodType.NumIn() >= 1 {
		firstParam := methodType.In(0)

		// 检查是否是 *Context 类型
		if firstParam.Kind() == reflect.Ptr && firstParam.Elem().Name() == "Context" {
			ctx := &Context{
				Response: w,
				Request:  r,
				InPath:   inPath,
			}
			args = []reflect.Value{reflect.ValueOf(ctx)}
		} else if firstParam.Kind() == reflect.Slice && firstParam.Elem().Kind() == reflect.String {
			// []string 参数 (兼容 SlightPHP 的 $inPath)
			args = []reflect.Value{reflect.ValueOf(inPath)}
		}
		// 如果参数类型不匹配，args 保持为空，方法调用会 panic（正常行为）
	} // NumIn() == 0: 无参数方法，args 保持空

	result := method.Call(args)

	// 处理返回值
	if len(result) > 0 {
		if err, ok := result[len(result)-1].Interface().(error); ok {
			return err
		}
		if data := result[0].Interface(); data != nil {
			if dataStr, ok := data.(string); ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(dataStr))
			}
		}
	}

	return nil
}

// toTitle 将字符串首字母转为大写（替代已废弃的 strings.Title）
func toTitle(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] -= 32
	}
	return string(runes)
}

func (s *SlightGo) handleNotFound(w http.ResponseWriter, r *http.Request, zone, page, entry string) {
	http.NotFound(w, r)
	log.Printf("[SlightGo] 404: /%s/%s/%s", zone, page, entry)
}

func (s *SlightGo) handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("[SlightGo] error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// 静态文件服务
// ---------------------------------------------------------------------------

// ServeStatic 注册静态文件路由
func (s *SlightGo) ServeStatic(urlPrefix, dir string) *SlightGo {
	fileServer := http.FileServer(http.Dir(dir))
	handler := http.StripPrefix(urlPrefix, fileServer)

	// 包装为中间件，优先匹配静态文件
	orig := s.middlewares
	s.middlewares = append(orig, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, urlPrefix) {
				handler.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	return s
}

// ---------------------------------------------------------------------------
// 辅助方法
// ---------------------------------------------------------------------------

// SetDefaultZone 设置默认 Zone
func (s *SlightGo) SetDefaultZone(zone string) { s.defaultZone = zone }

// SetDefaultPage 设置默认 Page
func (s *SlightGo) SetDefaultPage(page string) { s.defaultPage = page }

// SetDefaultEntry 设置默认 Entry
func (s *SlightGo) SetDefaultEntry(entry string) { s.defaultEntry = entry }

// DefaultZone 返回默认 Zone 名称
func (s *SlightGo) DefaultZone() string { return s.defaultZone }

// DefaultPage 返回默认 Page 名称
func (s *SlightGo) DefaultPage() string { return s.defaultPage }

// DefaultEntry 返回默认 Entry 名称
func (s *SlightGo) DefaultEntry() string { return s.defaultEntry }

// ---------------------------------------------------------------------------
// 便捷启动函数
// ---------------------------------------------------------------------------

// Run 是包级便捷函数，快速创建并启动应用
func Run(addr string, opts ...Option) {
	app := New(opts...)
	app.Run(addr)
}

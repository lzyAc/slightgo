package slightgo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// Context 封装 HTTP 请求的上下文信息，
// 在控制器方法中通过此对象获取请求数据和输出响应。
//
// 对应 SlightPHP 中 $inPath 参数的增强版本。
type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
	InPath   []string

	// 路由参数（来自 SRoute 自定义路由规则）
	Params map[string]string

	// 存储用户自定义数据（请求生命期内共享）
	store map[string]interface{}
}

// NewContext 创建一个新的 Context
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Response: w,
		Request:  r,
		InPath:   make([]string, 0),
		Params:   make(map[string]string),
		store:    make(map[string]interface{}),
	}
}

// ---------------------------------------------------------------------------
// 输入方法
// ---------------------------------------------------------------------------

// Get 获取 URL 查询参数
func (c *Context) Get(key string) string {
	return c.Request.URL.Query().Get(key)
}

// Post 获取 POST 表单参数
func (c *Context) Post(key string) string {
	return c.Request.FormValue(key)
}

// Query 获取 URL 查询参数（同 Get）
func (c *Context) Query(key string) string {
	return c.Get(key)
}

// Form 获取表单参数（同时支持 GET 和 POST）
func (c *Context) Form(key string) string {
	return c.Request.FormValue(key)
}

// Param 获取路由参数
func (c *Context) Param(key string) string {
	if c.Params != nil {
		return c.Params[key]
	}
	return ""
}

// SetParam 设置路由参数
func (c *Context) SetParam(key, value string) {
	if c.Params == nil {
		c.Params = make(map[string]string)
	}
	c.Params[key] = value
}

// BindJSON 将请求体中的 JSON 数据绑定到指定对象
func (c *Context) BindJSON(v interface{}) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	defer c.Request.Body.Close()
	return json.Unmarshal(body, v)
}

// BindForm 将表单数据绑定到指定对象（使用标准库的 url.Values 转换）
func (c *Context) BindForm(v interface{}) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// 输出方法
// ---------------------------------------------------------------------------

// JSON 输出 JSON 格式响应
func (c *Context) JSON(status int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(status)
	return json.NewEncoder(c.Response).Encode(data)
}

// String 输出纯文本响应
func (c *Context) String(status int, text string) error {
	c.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(status)
	_, err := c.Response.Write([]byte(text))
	return err
}

// HTML 输出 HTML 响应
func (c *Context) HTML(status int, html string) error {
	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response.WriteHeader(status)
	_, err := c.Response.Write([]byte(html))
	return err
}

// Redirect 重定向
func (c *Context) Redirect(status int, target string) error {
	if status == 0 {
		status = http.StatusFound // 302
	}
	http.Redirect(c.Response, c.Request, target, status)
	return nil
}

// NoContent 输出无内容响应
func (c *Context) NoContent(status int) error {
	c.Response.WriteHeader(status)
	return nil
}

// ---------------------------------------------------------------------------
// Header 操作
// ---------------------------------------------------------------------------

// SetHeader 设置响应头
func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

// GetHeader 获取请求头
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// ---------------------------------------------------------------------------
// Cookie 操作
// ---------------------------------------------------------------------------

// SetCookie 设置 Cookie
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response, cookie)
}

// GetCookie 获取 Cookie
func (c *Context) GetCookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// ---------------------------------------------------------------------------
// 数据存储
// ---------------------------------------------------------------------------

// Set 在上下文中存储数据
func (c *Context) Set(key string, value interface{}) {
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = value
}

// Get 从上下文中获取存储的数据
func (c *Context) Get(key string) (interface{}, bool) {
	if c.store == nil {
		return nil, false
	}
	v, ok := c.store[key]
	return v, ok
}

// ---------------------------------------------------------------------------
// 其他
// ---------------------------------------------------------------------------

// URL 获取完整 URL
func (c *Context) URL() *url.URL {
	return c.Request.URL
}

// Method 获取请求方法
func (c *Context) Method() string {
	return c.Request.Method
}

// Host 获取请求主机
func (c *Context) Host() string {
	return c.Request.Host
}

// RemoteAddr 获取客户端地址
func (c *Context) RemoteAddr() string {
	return c.Request.RemoteAddr
}

// UserAgent 获取 User-Agent
func (c *Context) UserAgent() string {
	return c.Request.UserAgent()
}

// ContentType 获取 Content-Type
func (c *Context) ContentType() string {
	return c.Request.Header.Get("Content-Type")
}

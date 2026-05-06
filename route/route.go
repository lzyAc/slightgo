// Package route 提供自定义路由规则
//
// 对应 SlightPHP 的 SRoute 插件。
// 支持在标准 Zone → Page → Entry 路由之外定义自定义路由规则，
// 支持参数化路径和正则匹配。
package route

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// 路由规则
// ---------------------------------------------------------------------------

// Rule 定义一条路由规则
type Rule struct {
	Method  string           // HTTP 方法（GET, POST, PUT, DELETE 等，空值匹配所有）
	Pattern string           // URL 模式，如 /user/:id
	regex   *regexp.Regexp   // 编译后的正则
	paramKeys []string        // 参数名列表
	Handler http.HandlerFunc // 处理函数
}

// Router 路由管理器
type Router struct {
	rules    []*Rule
	notFound http.HandlerFunc
}

// New 创建一个新的路由管理器
func New() *Router {
	return &Router{
		rules: make([]*Rule, 0),
		notFound: func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
	}
}

// ---------------------------------------------------------------------------
// 路由注册
// ---------------------------------------------------------------------------

// Get 注册 GET 路由
func (r *Router) Get(pattern string, handler http.HandlerFunc) *Router {
	return r.addRule("GET", pattern, handler)
}

// Post 注册 POST 路由
func (r *Router) Post(pattern string, handler http.HandlerFunc) *Router {
	return r.addRule("POST", pattern, handler)
}

// Put 注册 PUT 路由
func (r *Router) Put(pattern string, handler http.HandlerFunc) *Router {
	return r.addRule("PUT", pattern, handler)
}

// Delete 注册 DELETE 路由
func (r *Router) Delete(pattern string, handler http.HandlerFunc) *Router {
	return r.addRule("DELETE", pattern, handler)
}

// Patch 注册 PATCH 路由
func (r *Router) Patch(pattern string, handler http.HandlerFunc) *Router {
	return r.addRule("PATCH", pattern, handler)
}

// Any 注册匹配所有 HTTP 方法的路由
func (r *Router) Any(pattern string, handler http.HandlerFunc) *Router {
	return r.addRule("", pattern, handler)
}

// addRule 添加路由规则
func (r *Router) addRule(method, pattern string, handler http.HandlerFunc) *Router {
	rule := &Rule{
		Method:  strings.ToUpper(method),
		Pattern: pattern,
		Handler: handler,
	}

	// 将模式中的 :param 替换为命名捕获组
	regexStr, keys := compilePattern(pattern)
	rule.regex = regexp.MustCompile(regexStr)
	rule.paramKeys = keys

	r.rules = append(r.rules, rule)
	return r
}

// compilePattern 将 /user/:id 编译为正则表达式
func compilePattern(pattern string) (string, []string) {
	var keys []string
	var buf strings.Builder

	buf.WriteString("^")

	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		buf.WriteString("/")
		if strings.HasPrefix(part, ":") {
			// 参数占位符
			keys = append(keys, part[1:])
			buf.WriteString("([^/]+)")
		} else if strings.HasPrefix(part, "*") {
			// 通配符
			keys = append(keys, part[1:])
			buf.WriteString("(.*)")
		} else {
			// 字面量
			buf.WriteString(regexp.QuoteMeta(part))
		}
	}

	buf.WriteString("$")
	return buf.String(), keys
}

// ---------------------------------------------------------------------------
// 路由匹配
// ---------------------------------------------------------------------------

// Match 匹配 URL 路径和 HTTP 方法，返回匹配的路由规则和参数
// 如果未匹配到，返回 nil
func (r *Router) Match(method, path string) (*Rule, map[string]string) {
	method = strings.ToUpper(method)

	for _, rule := range r.rules {
		// 方法匹配
		if rule.Method != "" && rule.Method != method {
			continue
		}

		// 路径匹配
		matches := rule.regex.FindStringSubmatch(path)
		if matches == nil {
			continue
		}

		// 提取参数
		params := make(map[string]string)
		for i, key := range rule.paramKeys {
			if i+1 < len(matches) {
				params[key] = matches[i+1]
			}
		}

		return rule, params
	}

	return nil, nil
}

// ServeHTTP 实现 http.Handler 接口
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rule, params := r.Match(req.Method, req.URL.Path)
	if rule == nil {
		r.notFound(w, req)
		return
	}

	// 将参数放入请求上下文（通过 Header 传递，或直接修改请求）
	log.Printf("[Route] matched: %s %s → params=%v", req.Method, req.URL.Path, params)

	// 执行处理函数
	rule.Handler(w, req)
}

// SetNotFound 设置 404 处理函数
func (r *Router) SetNotFound(handler http.HandlerFunc) {
	r.notFound = handler
}

// ---------------------------------------------------------------------------
// 辅助方法
// ---------------------------------------------------------------------------

// URL 根据路由名称和参数生成 URL（简化实现）
// 模式: /user/:id  +  params{"id": "123"} → /user/123
func (r *Router) URL(pattern string, params map[string]string) string {
	result := pattern
	for k, v := range params {
		result = strings.ReplaceAll(result, ":"+k, v)
	}
	return result
}

// Print 打印所有路由规则（调试用）
func (r *Router) Print() {
	fmt.Println("[Route] registered rules:")
	for _, rule := range r.rules {
		method := rule.Method
		if method == "" {
			method = "ANY"
		}
		fmt.Printf("  %s %s\n", method, rule.Pattern)
	}
}

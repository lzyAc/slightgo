# SlightGo

**SlightPHP 的 Go 语言版本** — 轻量级、高效的 Go Web 开发框架

[![Go Version](https://img.shields.io/badge/Go-1.22+-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-lzyAc%2Fslightgo-blue)](https://github.com/lzyAc/slightgo)

---

## 概述

SlightGo 是 [SlightPHP](https://github.com/hetao29/slightphp) (hetao29 开发的高效 PHP 敏捷开发框架) 的 Go 语言移植版本。沿用了 SlightPHP 的核心设计理念：**框架与插件分离**、**Zone → Page → Entry 三层路由架构**，并利用 Go 语言的特性提供了更好的性能和类型安全。

### 设计理念

| 概念 | 说明 | 默认值 |
|------|------|--------|
| **Zone** | 模块/目录名 | `"zone"` |
| **Page** | 控制器/文件名 | `"page"` |
| **Entry** | 方法名 (`Page` 前缀) | `"entry"` |
| **splitFlag** | URL 分隔符 | `"/"` |

### URL 解析

```
http://localhost:8080/{zone}/{page}/{entry}
```

访问 `http://localhost:8080/user/profile/show` 将调用 `UserPage.PageShow(inPath)` 方法。

---

## 快速开始

### 安装

```bash
go get github.com/lzyAc/slightgo
```

### Hello World

```go
package main

import "github.com/lzyAc/slightgo"

type IndexPage struct{}

func (p *IndexPage) PageEntry(ctx *slightgo.Context) {
	ctx.String(200, "Hello, SlightGo!")
}

func main() {
	app := slightgo.New()
	app.Zone("zone", func(z *slightgo.Zone) {
		z.Page("page", &IndexPage{})
	})
	app.Run(":8080")
}
```

访问 `http://localhost:8080` 即可看到输出。

### 带参数的示例

```go
package main

import (
	"github.com/lzyAc/slightgo"
	"github.com/lzyAc/slightgo/middleware"
)

type UserPage struct{}

// 对应 URL: /user/profile/{entry}
func (p *UserPage) PageShow(ctx *slightgo.Context) {
	ctx.JSON(200, map[string]interface{}{
		"zone":  ctx.InPath[0],
		"page":  ctx.InPath[1],
		"entry": ctx.InPath[2],
	})
}

func main() {
	app := slightgo.New(
		slightgo.WithDefaultZone("user"),
		slightgo.WithDefaultPage("profile"),
		slightgo.WithDefaultEntry("show"),
	)

	app.Use(middleware.Logger)
	app.Use(middleware.Recovery)

	app.Zone("user", func(z *slightgo.Zone) {
		z.Page("profile", &UserPage{})
	})

	app.Run(":8080")
}
```

---

## 架构

### 核心包

```
slightgo/
├── slightgo.go       # 核心框架 — 路由调度、HTTP 服务器
├── context.go        # 请求上下文 — 输入输出封装
├── config.go         # 配置管理入口
├── config/           # 配置加载（JSON）
├── db/               # 数据库 — 读写分离、连接池
├── cache/            # 缓存 — Memory / Redis
├── redis/            # Redis 客户端封装
├── tpl/              # 模板引擎（Go html/template）
├── route/            # 自定义路由规则（SRoute）
├── log/              # 日志分级
└── middleware/       # 常用中间件
```

### 对应关系

| SlightPHP | SlightGo | 说明 |
|-----------|----------|------|
| SlightPHP::run() | slightgo.New().Run() | 框架入口 |
| Zone | Zone (模块) | URL 第一段 |
| Page | Page (控制器 struct) | URL 第二段 |
| Entry | Page* 方法 | URL 第三段 |
| $inPath | ctx.InPath | URL 路径参数 |
| SDb | db 包 | 数据库（读写分离） |
| SRoute | route 包 | 自定义路由 |
| STpl | tpl 包 | 模板引擎 |
| SCache | cache 包 | 缓存 |
| SRedis | redis 包 | Redis |
| SConfig | config 包 | 配置 |
| SError | log 包 | 日志 |

---

## 详细文档

### 路由 (Zone → Page → Entry)

SlightGo 的核心路由机制：

```go
app.Zone("admin", func(z *slightgo.Zone) {
    z.Page("user", &AdminUserController{})
    z.Page("post", &AdminPostController{})
})
```

URL 映射：
- `GET /admin/user/list` → `AdminUserController.PageList(ctx)`
- `GET /admin/post/edit` → `AdminPostController.PageEdit(ctx)`

### Context

`Context` 封装了 HTTP 请求和响应：

```go
// 输入
ctx.Get("name")          // URL 查询参数 ?name=xxx
ctx.Post("name")         // POST 表单参数
ctx.Param("id")          // 路由参数（配合 route 包）
ctx.BindJSON(&obj)       // 解析 JSON 请求体

// 输出
ctx.JSON(200, data)      // JSON 响应
ctx.String(200, "text")  // 文本响应
ctx.HTML(200, "<h1>Hi</h1>")  // HTML 响应
ctx.Redirect(302, "/")   // 重定向

// 请求信息
ctx.Method()             // HTTP 方法
ctx.Host()               // 主机名
ctx.UserAgent()          // User-Agent
ctx.InPath               // URL 路径段数组
```

### 数据库 (读写分离)

```go
import "github.com/lzyAc/slightgo/db"

cfg := &db.Config{
    Driver: "mysql",
    DSN:    "user:pass@tcp(master:3306)/dbname",
    Reads:  []string{
        "user:pass@tcp(slave1:3306)/dbname",
        "user:pass@tcp(slave2:3306)/dbname",
    },
}

database, _ := db.New(cfg)
database.Query("SELECT * FROM users")    // 自动路由到从库
database.Exec("UPDATE users SET ...")    // 路由到主库
database.Transaction(func(tx *sql.Tx) error {
    // 事务操作
    return nil
})
```

### 缓存

```go
import (
    "github.com/lzyAc/slightgo/cache"
    "github.com/lzyAc/slightgo/redis"
)

// 内存缓存
mc := cache.NewMemory()
mc.Set("key", "value", 5*time.Minute)
v, ok := mc.Get("key")

// Redis 缓存
client := redis.New(&redis.Options{
    Addr: "127.0.0.1:6379",
})
rc := cache.NewRedis(client, "app:prefix")
rc.Set("key", "value", 5*time.Minute)
```

### 模板引擎

```go
import "github.com/lzyAc/slightgo/tpl"

engine := tpl.New(tpl.WithDir("templates"), tpl.WithExt(".html"))

// 渲染模板文件
html, _ := engine.Fetch("index", map[string]interface{}{
    "title": "Hello",
})

// 渲染字符串模板
html, _ = engine.Render("greeting", "Hello, {{.name}}!", map[string]interface{}{
    "name": "World",
})
```

### 自定义路由 (SRoute)

```go
import "github.com/lzyAc/slightgo/route"

r := route.New()
r.Get("/user/:id", func(w http.ResponseWriter, req *http.Request) {
    // 处理请求
})
r.Post("/api/create", handler)
```

### 中间件

```go
import "github.com/lzyAc/slightgo/middleware"

app.Use(middleware.Logger)    // 请求日志
app.Use(middleware.Recovery)  // Panic 恢复
app.Use(middleware.CORS(nil)) // 跨域支持
```

### 配置

```go
import "github.com/lzyAc/slightgo/config"

// 加载 JSON 配置文件
config.Load("config.json")
host := config.GetString("server.host")

// 或使用框架封装
app.LoadConfig("config.json")
```

---

## 完整示例

参见 [examples/demo](examples/demo/) 目录。

```bash
cd examples/demo
go run main.go
```

然后访问 http://localhost:8080。

---

## 与 SlightPHP 的差异

1. **类型安全**：Go 是静态类型语言，所有方法签名在编译时检查
2. **并发模型**：Go 原生支持 goroutine，无需额外配置即可处理高并发
3. **编译部署**：编译为单一二进制文件，部署简单
4. **无动态加载**：Go 编译后不可动态加载代码，因此 Zone/Page 需要显式注册而非自动发现
5. **模板引擎**：使用 Go 标准库 html/template，默认自动转义 XSS

---

## 版本

**v0.1.0** — 初始版本

---

## 许可证

MIT License

---

## 相关链接

- [SlightGo GitHub](https://github.com/lzyAc/slightgo)
- [SlightPHP (PHP 原版)](https://github.com/hetao29/slightphp)
- [Go 标准库](https://golang.org/pkg/)

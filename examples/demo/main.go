// Demo 示例应用
//
// 展示 SlightGo 框架的基本用法：
// 1. Zone → Page → Entry 路由
// 2. Context 使用
// 3. 中间件
// 4. 配置加载
// 5. 数据库操作
package main

import (
	"fmt"

	"github.com/hetao29/slightgo"
	"github.com/hetao29/slightgo/middleware"
)

// ---------------------------------------------------------------------------
// 控制器（Page）
// ---------------------------------------------------------------------------

// UserPage 用户相关控制器
// URL: /user/profile/:entry
type UserPage struct{}

// PageIndex 对应 entry=index
// URL: /user/profile/index 或 /user/profile
func (p *UserPage) PageIndex(ctx *slightgo.Context) {
	ctx.HTML(200, "<h1>User Profile</h1><p>Welcome to user profile page!</p>")
}

// PageShow 对应 entry=show
// URL: /user/profile/show
func (p *UserPage) PageShow(ctx *slightgo.Context) {
	ctx.JSON(200, map[string]interface{}{
		"zone":   ctx.InPath[0],
		"page":   ctx.InPath[1],
		"entry":  ctx.InPath[2],
		"params": ctx.InPath[3:],
	})
}

// HomePage 首页控制器
// URL: /
type HomePage struct{}

// PageEntry 默认 entry
func (p *HomePage) PageEntry(ctx *slightgo.Context) {
	ctx.HTML(200, `
		<html>
		<head><title>SlightGo Demo</title></head>
		<body>
			<h1>SlightGo Framework</h1>
			<p>SlightPHP 的 Go 语言版本</p>
			<ul>
				<li><a href="/user/profile/index">User Profile</a></li>
				<li><a href="/user/profile/show">User Profile (JSON)</a></li>
				<li><a href="/home/info">Info</a></li>
			</ul>
		</body>
		</html>
	`)
}

// InfoPage 信息控制器
type InfoPage struct{}

// PageInfo 版本信息
func (p *InfoPage) PageInfo(ctx *slightgo.Context) {
	ctx.String(200, fmt.Sprintf("SlightGo v0.1.0\nPath: %v\nMethod: %s", ctx.InPath, ctx.Method()))
}

// ---------------------------------------------------------------------------
// 主函数
// ---------------------------------------------------------------------------

func main() {
	// 使用选项创建应用
	app := slightgo.New(
		slightgo.WithDefaultZone("home"),
		slightgo.WithDefaultPage("home"),
		slightgo.WithDefaultEntry("entry"),
	)

	// 注册中间件
	app.Use(middleware.Logger)
	app.Use(middleware.Recovery)

	// 注册路由 - Zone "home"
	app.Zone("home", func(z *slightgo.Zone) {
		z.Page("home", &HomePage{})
		z.Page("info", &InfoPage{})
	})

	// 注册路由 - Zone "user"
	app.Zone("user", func(z *slightgo.Zone) {
		z.Page("profile", &UserPage{})
	})

	// 加载配置
	cfg, err := app.LoadConfig("config.json")
	if err != nil {
		fmt.Println("Warning: config.json not found, using defaults")
	} else {
		host := cfg.GetString("server.host")
		port := cfg.GetString("server.port")
		fmt.Printf("Config loaded: %s:%s\n", host, port)
	}

	// 启动服务器
	app.Run(":8080")
}

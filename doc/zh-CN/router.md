## Router 组件

Router 用于解析 Http Request URL 规则，并选择匹配的 Module:Controller/Action 并执行。

与大多数 Web 框架不同，httpsrv 弱化了 router 功能，只包含两种最简单的路由类型：固定路由和静态路由。

* **固定路由**（或称标准路由）：除了注册 Module 到 Service 时指定的 baseuri 参数可变以外，其后面的路径都是两级固定的（controller/action 名）
* **静态路由**：用于单独设置静态文件的访问规则（参考 [Module 组件](module.md)）

## 固定路由规则

固定路由是系统默认，不需要单独设置，其在 URL 对应的名称严格与 Controller/Action 的名称对应：

### 基本规则

1. 控制器里面的 Action 方法必需以 `*Action()` 结尾才能被路由匹配并执行
2. 访问服务时，URL 对应的名称统一转换为小写
3. 名称包含驼峰写法，用 `-` 符号连接，如 `NameUpper` 对应 URL 名称是 `name-upper`
4. "Index" 是默认名：
   - 当 Action 名为 `IndexAction()` 时，在 URL 中可省略 Action 名
   - 当 Controller 和 Action 同时为 `type Index struct` 和 `IndexAction()` 时，可同时省略 URL 两级的名称

### URL 命名约定

| Controller/Action 名称 | URL 路径 |
|---|---|
| `type User struct` + `LoginAction()` | `/user/login` |
| `type User struct` + `IndexAction()` | `/user` |
| `type Index struct` + `IndexAction()` | `/` |
| `type UserProfile struct` + `EditAction()` | `/user-profile/edit` |
| `type API struct` + `GetUserInfoAction()` | `/api/get-user-info` |

## 路由匹配示例

### 示例 1：基础路由

``` go
type User struct {
	*httpsrv.Controller
}

func (c User) LoginAction() {
	c.RenderString("登录页面")
}

func (c User) RegisterAction() {
	c.RenderString("注册页面")
}

func (c User) ProfileAction() {
	c.RenderString("用户资料")
}

func (c User) IndexAction() {
	c.RenderString("用户列表")
}
```

访问路径：

* `/user/login` → User.LoginAction()
* `/user/register` → User.RegisterAction()
* `/user/profile` → User.ProfileAction()
* `/user` 或 `/user/index` → User.IndexAction()

### 示例 2：带 Module 前缀的路由

``` go
// main.go
func main() {
	// 注册 Module 到 /api/v1 路径
	httpsrv.GlobalService.HandleModule("/api/v1", NewApiModule())
	
	httpsrv.GlobalService.Config.HttpPort = 8080
	httpsrv.GlobalService.Start()
}
```

``` go
// api 模块
type ApiController struct {
	*httpsrv.Controller
}

func (c ApiController) UserAction() {
	c.RenderJson(map[string]string{
		"message": "用户信息",
	})
}

func (c ApiController) ProductAction() {
	c.RenderJson(map[string]string{
		"message": "产品信息",
	})
}
```

访问路径：

* `/api/v1/api/user` → ApiController.UserAction()
* `/api/v1/api/product` → ApiController.ProductAction()

### 示例 3：Index 默认路由

``` go
type Index struct {
	*httpsrv.Controller
}

func (c Index) IndexAction() {
	c.RenderString("首页")
}

type Dashboard struct {
	*httpsrv.Controller
}

func (c Dashboard) IndexAction() {
	c.RenderString("仪表盘首页")
}

func (c Dashboard) SettingsAction() {
	c.RenderString("仪表盘设置")
}
```

访问路径：

* `/` → Index.IndexAction()（省略 Controller 和 Action）
* `/index` → Index.IndexAction()（省略 Action）
* `/dashboard` → Dashboard.IndexAction()（省略 Action）
* `/dashboard/settings` → Dashboard.SettingsAction()

### 示例 4：驼峰命名转换

``` go
type UserProfile struct {
	*httpsrv.Controller
}

func (c UserProfile) EditAction() {
	c.RenderString("编辑用户资料")
}

func (c UserProfile) UpdateSettingsAction() {
	c.RenderString("更新设置")
}
```

访问路径：

* `/user-profile/edit` → UserProfile.EditAction()
* `/user-profile/update-settings` → UserProfile.UpdateSettingsAction()

## 模块挂载与路由

### 单个模块挂载到根路径

``` go
func main() {
	// 模块直接挂载到根路径
	httpsrv.GlobalService.HandleModule("/", NewWebModule())
	
	httpsrv.GlobalService.Config.HttpPort = 8080
	httpsrv.GlobalService.Start()
}
```

访问路径：`/controller/action`

### 多个模块挂载到不同路径

``` go
func main() {
	// 前端模块挂载到根路径
	httpsrv.GlobalService.HandleModule("/", NewFrontendModule())
	
	// API 模块挂载到 /api/v1
	httpsrv.GlobalService.HandleModule("/api/v1", NewApiModule())
	
	// 管理后台挂载到 /admin
	httpsrv.GlobalService.HandleModule("/admin", NewAdminModule())
	
	httpsrv.GlobalService.Config.HttpPort = 8080
	httpsrv.GlobalService.Start()
}
```

访问路径：

* 前端：`/controller/action`
* API：`/api/v1/controller/action`
* 管理后台：`/admin/controller/action`

### 模块内嵌套结构

``` go
// api/v1 模块
type UserController struct {
	*httpsrv.Controller
}

func (c UserController) ListAction() {
	c.RenderJson([]string{"user1", "user2"})
}

type ProductController struct {
	*httpsrv.Controller
}

func (c ProductController) ListAction() {
	c.RenderJson([]string{"product1", "product2"})
}

func NewApiModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(UserController), new(ProductController))
	return mod
}
```

注册到 Service：

``` go
httpsrv.GlobalService.HandleModule("/api/v1", NewApiModule())
```

访问路径：

* `/api/v1/user/list` → UserController.ListAction()
* `/api/v1/product/list` → ProductController.ListAction()

## 静态文件路由

静态文件通过 Module 的 `RegisterFileServer` 方法注册：

``` go
func NewModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	
	// 注册静态文件服务器
	// 第一个参数：URL 访问路径前缀
	// 第二个参数：本地文件系统路径
	// 第三个参数：可选的配置（如是否启用目录列表等）
	mod.RegisterFileServer("/assets", "/path/of/static/files", nil)
	
	return mod
}
```

访问规则：

``` go
// Module 挂载到 /cms
httpsrv.GlobalService.HandleModule("/cms", ui.NewModule())

// 静态文件路径：/path/of/static/files/js/main.js
// 访问 URL：/cms/assets/js/main.js
```

完整的 URL 组合公式：

```
final_url = baseuri + static_prefix + file_path
```

示例：

``` go
// 注册
mod.RegisterFileServer("/assets", "./static", nil)

// 文件系统结构
./static/
├── css/
│   └── main.css
├── js/
│   └── app.js
└── img/
    └── logo.png

// 访问 URL
/cms/assets/css/main.css
/cms/assets/js/app.js
/cms/assets/img/logo.png
```

## 路由优先级

当有多个路由规则可能匹配同一个 URL 时，httpsrv 按照以下优先级顺序匹配：

1. **静态文件路由**优先级最高
2. **Module 路由**按照注册顺序匹配，第一个匹配的生效

``` go
func main() {
	// 先注册静态文件
	httpsrv.GlobalService.HandleModule("/", NewStaticModule())
	
	// 再注册动态路由
	httpsrv.GlobalService.HandleModule("/api", NewApiModule())
}
```

## 完整示例

### 项目结构

``` go
package main

import "github.com/hooto/httpsrv"

// 前端控制器
type Index struct {
	*httpsrv.Controller
}

func (c Index) IndexAction() {
	c.Data["title"] = "首页"
	c.Render()
}

type About struct {
	*httpsrv.Controller
}

func (c About) IndexAction() {
	c.Data["title"] = "关于我们"
	c.Render()
}

// API 控制器
type ApiController struct {
	*httpsrv.Controller
}

func (c ApiController) UserAction() {
	c.RenderJson(map[string]string{
		"status": "ok",
		"data":   "用户信息",
	})
}

// 管理后台控制器
type AdminController struct {
	*httpsrv.Controller
}

func (c AdminController) DashboardAction() {
	c.RenderString("管理后台仪表盘")
}

// 模块构建
func NewFrontendModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.SetTemplatePath("./views/frontend")
	mod.RegisterController(new(Index), new(About))
	mod.RegisterFileServer("/static", "./static", nil)
	return mod
}

func NewApiModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(ApiController))
	return mod
}

func NewAdminModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.SetTemplatePath("./views/admin")
	mod.RegisterController(new(AdminController))
	return mod
}

func main() {
	// 注册多个模块
	httpsrv.GlobalService.HandleModule("/", NewFrontendModule())
	httpsrv.GlobalService.HandleModule("/api/v1", NewApiModule())
	httpsrv.GlobalService.HandleModule("/admin", NewAdminModule())
	
	httpsrv.GlobalService.Config.HttpPort = 8080
	httpsrv.GlobalService.Start()
}
```

访问路径映射：

| URL | 匹配 | 说明 |
|---|---|---|
| `/` | Index.IndexAction() | 首页 |
| `/about` | About.IndexAction() | 关于页面 |
| `/static/css/style.css` | 静态文件 | CSS 文件 |
| `/api/v1/api/user` | ApiController.UserAction() | API 用户信息 |
| `/admin/admin/dashboard` | AdminController.DashboardAction() | 管理后台仪表盘 |

## 注意事项

1. **Action 命名**：Action 方法必须以 `Action()` 结尾，否则不会被路由识别
2. **URL 转换**：Controller 和 Action 名称会自动转换为小写，驼峰命名用 `-` 连接
3. **Index 特殊性**：名为 `Index` 的 Controller 或 Action 可以在 URL 中省略
4. **模块路径**：Module 的 baseuri 必须以 `/` 开头，但不要以 `/` 结尾
5. **静态文件**：静态文件路径会与 Module 的 baseuri 组合，注意避免路径冲突
6. **路由冲突**：避免不同模块的 baseuri 路径重叠，可能导致路由匹配混乱

## 路由最佳实践

1. **命名规范**：Controller 和 Action 使用有意义的英文命名，避免中文拼音
2. **RESTful 风格**：API 模块建议使用资源命名方式，如 UserController 的 GetAction、PostAction、PutAction、DeleteAction
3. **模块划分**：按业务功能划分模块，如 `/api/user`、`/api/product`、`/admin/system`
4. **静态资源**：统一使用 `/static` 或 `/assets` 前缀管理静态资源
5. **版本控制**：API 模块建议在路径中包含版本号，如 `/api/v1`、`/api/v2`

## 与其他框架的对比

| 框架 | 路由类型 | 是否支持正则 | 是否支持参数提取 |
|---|---|---|---|
| httpsrv | 固定路由 | 否 | 否 |
| Beego | 支持多种路由 | 是 | 是 |
| Gin | 支持 RESTful 路由 | 是 | 是 |
| Revel | 支持多种路由 | 是 | 是 |

httpsrv 选择了最简单固定的路由方式，这是为了：

1. **保持简洁**：减少配置复杂度，提高代码可读性
2. **性能优化**：固定路由匹配速度更快
3. **约定优于配置**：通过命名约定规范 URL 结构
4. **降低学习成本**：开发者无需学习复杂的路由规则

如果需要更复杂的路由功能，可以在 Controller 中通过 `c.Params` 手动解析 URL 参数，或使用自定义的 Filter 实现。
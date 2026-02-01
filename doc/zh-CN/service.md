## Service 组件

Service 组件是 httpsrv 提供HTTP服务的最外层对象实例, 其它组件都直接或者间接的注册到 Service 以后，最后通过 Service.Start() 方法对外提供 HTTP 服务.

> 提示: 由外至内 httpsrv 主要组件的逻辑层依次是: HTTP 请求 -> Service -> Module -> Controller -> Action

``` go
type Service struct {
	Config         Config
	Filters        []Filter
	TemplateLoader *TemplateLoader
}
```

说明

| 项目 | 说明 |
|----|----|
| Config | 基础配置组件, 定义 HTTP 服务启动时的依赖参数, [Config 详情](config.md) |
| Filter | 以 HTTP Request/Response 整个执行生命周期内的过滤器序列配置, httpsrv 以此顺序执行如 Router, Params, Action 等核心逻辑. 这是一个抽象接口定义，可定制，但多数情况下无需配置，系统默认设置已经满足多数使用场景. 默认配置参考 [文件 filter.go](https://github.com/hooto/httpsrv/blob/master/filter.go)  |
| TemplateLoader | 视图加载管理组件, 当开发 Web MVC 中的 V(View) 时会自动激活这个组件，具体可参考 [Template 详情](template.md) |

## 快速使用 Service

httpsrv 默认创建了一个全局 Service 实例 `httpsrv.GlobalService`, 多数场景下直接使用它即可. 最简单的示例，只需要注册一个 module/controller 并设置 tcp 端口就可启用服务, 如:


``` go
package main

import (
	"github.com/hooto/httpsrv"
)

type Account struct {
	*httpsrv.Controller
}

func (c Account) LoginAction() {
	c.RenderString("hello world")
}

// 构建一个模块示例
func NewUserModule() *httpsrv.Module {

	// 初始化一个空的模块
	mod := httpsrv.NewModule()
    
	// 注册一个控制器到模块中
	mod.ControllerRegister(new(Account))

	return mod
}

func main() {

	// 将这个模块注册到服务里 (挂载到 URL 路径为 /user 对外提供服务, 可配置)
	httpsrv.GlobalService.HandleModule("/user", NewUserModule())

	// 设置服务端口
	httpsrv.GlobalService.Config.HttpPort = 8080

	// 启动服务
	httpsrv.GlobalService.Start()
}
```

编译并启动服务

``` shell
go build -o demo-server main.go
./demo-server
```

按照 `/{module-path}/{controller}/{action}` 的全局名称约定，以上服务通过 http://localhost:8080/user/account/login 访问.


## 多 Service 并存

在部分场景中，需要对外以不同端口提供多组服务实例. 比如: 80端口为前端业务(企业防火墙只开放 80 端口), 8080端口为API业务并只在内网开放, 可以如下实现:  

``` go
package main

import (
	"github.com/hooto/httpsrv"
)

type ApiDemo struct {
	*httpsrv.Controller
}

func (c ApiDemo) ExampleAction() {
	jsonStruct := struct {
		Name string `json:"name"`
	} {
		Name: "robot"
	}
	c.RenderJson(jsonStruct)
}

// 构建API模块
func NewApiModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(ApiDemo))
	return mod
}

type Index struct {
	*httpsrv.Controller
}

func (c Index) IndexAction() {
	c.RenderString("hello world")
}

// 构建前端模块
func NewFrontendModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(Index))
	return mod
}

func main() {

	serviceFrontend := httpsrv.NewService()
	serviceFrontend.Config.HttpPort = 80
	serviceFrontend.HandleModule("/", NewFrontendModule())

	serviceApi := httpsrv.NewService()
	serviceApi.Config.HttpPort = 8080
	serviceApi.HandleModule("/api/v1", NewApiModule())


	// 启动前端服务
	go serviceFrontend.Start()

	// 启动后端服务
	serviceApi.Start()
}
```


## Service 主要接口方法

在 `type Service struct` 这个数据定义基础之上，扩展了部分动态接口方法用于定制配置项

### 核心方法 ModuleRegister

``` go
// 接口定义
func (s *Service) HandlerRegister(baseuri string, h http.Handler)
```

这是一个必需的方法，所有模块都需要注册到 Service 上才能提供对外服务, 具体示例可参考如上代码. 

需要特别说明 baseuri string 这个参数, 在多数企业应用中，基于工程和业务需求的考虑，都会有多个模块共存的场景 (即: 同域名下有多个业务系统)，不同模块在注册到 Service 上时，都需要指定这个模块在 URL 的挂载目录，baseuri 路径名一般和业务系统功能对应，比如 /user, /cms, /mail 等等.  


### HandlerRegister, HandlerFuncRegister

``` go
// 接口定义 http.Handler
func (s *Service) HandlerRegister(baseuri string, h http.Handler)

// 接口定义 http.HandlerFunc
func (s *Service) HandlerFuncRegister(baseuri string, h http.HandlerFunc)
```

这两个接口用于向 Service 注册原生的 go/net/http 处理函数, 主要用于:

* RPC 类处理函数
* WebSocket 类处理函数


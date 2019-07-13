## Module 组件

Module 用于组合具有相似业务功能的代码, 包含 Controller, Router, Template 等相关组件.

### Module 的典型使用

假设一个 API 服务, 模块名为 github.com/hooto/httpsrv-demo/websrv/v1 , 它包含 User, Role 两个 Controller, 一般我们在这个模块的根目录新建一个 setup.go 文件，并导入涉及的这两个控制器，如: 


``` go
// vim setup.go
package v1

import (
	"github.com/hooto/httpsrv"
)

func NewModule() httpsrv.Module {

	module := httpsrv.NewModule("api-v1")
    
	module.ControllerRegister(new(User))
	module.ControllerRegister(new(Role))

	return module
}

// User, Role 具体代码实现省略...
```

在 main.go 入口文件里注册上述模块到服务(Service) 中, 并通过 URL /api/v1 前缀访问模块内的各个方法:

``` go
package main

import (
	"github.com/hooto/httpsrv"
	"github.com/hooto/httpsrv-demo/websrv/v1"
)

func main() {

	httpsrv.GlobalService.ModuleRegister("/api/v1", v1.NewModule())

	httpsrv.GlobalService.Config.HttpPort = 8080

	httpsrv.GlobalService.Start()
}
```

### 设置 View Template 路径

如果当前 Module 是一个前端UI类业务模块，需要视图模版，通过如下接口设置模版路径:

``` go
func NewModule() httpsrv.Module {
	module := httpsrv.NewModule("ui")
	module.TemplatePathSet("/path/of/module/ui/views") // 模版本地文件的路径, 可谓一个模块设置 1 ~ N 个路径
	return module
}
```

### 设置静态文件路径

如果当前 Module 依赖静态文件，如 js, css, img 等, 通过如下接口设置路径

``` go
func NewModule() httpsrv.Module {
	module := httpsrv.NewModule("ui")

	module.RouteSet(httpsrv.Route{
		Type:       httpsrv.RouteTypeStatic,
		Path:       "assets",
		StaticPath: "/path/of/static/files",
	})

	return module
}
```

注: 

Route{Path: "assets"} 参数表示前端访问的 URL 路径为 Service.ModuleRegister(baseuri,...) 设置的 baseuri + "/assets/静态文件相对路径", 如:

``` go
httpsrv.GlobalService.ModuleRegister("/cms", ui.NewModule())
```

静态文件相对路径为 js/main.js, 则最终的 URL 静态文件访问路径是:

``` shell
# baseuri + route.path + static.file
/cms/assets/js/main.js
```


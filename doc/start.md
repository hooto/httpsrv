## 系统环境准备

httpsrv 是 Golang HTTP 框架, 需确保已经正常安装和配置 go 开发环境, 详情参考官方网站: 

* 官方原站 [https://golang.org](https://golang.org) 
* 中文镜像 [https://golang.google.cn](https://golang.google.cn)

> 注: 推荐在 Linux, Unix 或 MacOS 系统上开发和部署基于 httpsrv 的应用程序 (Windows OS 在企业应用中并不常见，所以没做兼容测试). 


安装，或更新 httpsrv:

``` shell
go get -u github.com/hooto/httpsrv
```

如上述步骤没有异常，开始一个简单的示例

## 开始一个 "Hello World" 示例

创建一个文件 main.go 并编辑:

``` go
package main

import (
	"github.com/hooto/httpsrv"
)

// 新建一个 Controller
type Hello struct {
	*httpsrv.Controller
}

func (c Hello) WorldAction() {
	c.RenderString("hello world")
}

// 构建一个模块示例
func NewModule() httpsrv.Module {

	// 初始化一个空的模块
	module := httpsrv.NewModule("demo")
    
	// 注册一个控制器到模块中
	module.ControllerRegister(new(Hello))

	return module
}

// 全局入口
func main() {

	// 将模块注册到服务(Service)容器中
	httpsrv.GlobalService.ModuleRegister("/", NewModule())

	// 设置服务端口
	httpsrv.GlobalService.Config.HttpPort = 8080

	// 启动服务
	httpsrv.GlobalService.Start()
}
```

运行这个示例:
``` shell
go run main.go
```

通过 http://localhost:8080/hello/world/ 便可访问上面新建的服务


## 开始一个正式的项目

上述的 "hello world" 只是一个简单示例，实际项目会更复杂:

### 目录结构建议:

作为面向模块编程的 HTTP 框架, 建议为每个模块(Module)业务逻辑新建独立的文件目录来组织管理代码, 使项目整体结构清晰, 管理便利, 如示例.

``` shell
├─ bin/
│  └─ cmd-server // 编译后的可执行文件 go build -o bin/cmd-server cmd/server/main.go
├─ etc/
│  └─ config.ini,json,yaml, ... // 配置文件存放
├─ config/
│  └─ config.go // 配置文件解析
├─ cmd/
│  └─ server/
│     └─ main.go // Server 服务入口
├─ data/
│  └─ data.go // 数据库, 存储访问 (MySQL, PostgreSQL, Redis, ...)
├─ websrv/ // 模块
│  ├─ api-v1/ // 模块 api-v1
│  │  ├─ setup.go
│  │  └─ controller-a.go
│  └─ frontend/ // 模块 frontend
│     ├─ setup.go
│     ├─ controller-a.go
│     └─ views/ // HTML视图模版文件
│        └─ controller-a/
│           ├─ action-a.tpl
│           └─ action-b.tpl
├─ webui/ // 静态文件模块
│  └─ a/ // 静态文件模块 a
│     ├─ img/*
│     ├─ css/*
│     └─ js/*
└─ var/
```

> 注: golang 有特定的 import 路径规则, 需确保目录正确


以上目录结构完整代码示例可参考 [https://github.com/hooto/httpsrv-demo](https://github.com/hooto/httpsrv-demo), 可以git导出并运行:

``` shell
# 使用 git 导出示例代码
git clone git@github.com:hooto/httpsrv-demo.git
cd httpsrv-demo

# 运行示例
./develop-run.sh 
I 2019-07-13 17:21:37.643316 config.go:29] project prefix path /opt/gopath/src/github.com/hooto/httpsrv-demo
I 2019-07-13 17:21:37.644743 service.go:240] lessgo/httpsrv: listening on tcp/0.0.0.0:8080
```

通过 http://localhost:8080/ 便可访问上面示例中的服务, 你如果开始一个全新项目，此示例代码可作为为模版使用。

### 业务逻辑依赖库

httpsrv 是纯粹精简的 http 框架，只封装 http request/response 有关的业务常用接口，对于 Web 开发中涉及的 Model, ORM, Cache 等业务中间件没有内置提供，而是根据需求引用第三方库.

* [https://github.com/lynkdb/mysqlgo](https://github.com/lynkdb/mysqlgo) Go client for MySQL
* [https://github.com/lynkdb/pgsqlgo](https://github.com/lynkdb/pgsqlgo) Go client for PostgreSQL
* [https://github.com/lynkdb/redisgo](https://github.com/lynkdb/redisgo) Go client for Redis
* [https://github.com/lynkdb/ssdbgo](https://github.com/lynkdb/ssdbgo) Go client for SSDB
* [https://github.com/lynkdb/kvgo](https://github.com/lynkdb/kvgo) An embedded Key-Value database library for Go language
* [https://github.com/hooto/hlog4g](https://github.com/hooto/hlog4g) Log library for Golang
* [https://github.com/hooto/hini4g](https://github.com/hooto/hini4g) INI file read library for Golang
* [https://github.com/hooto/hflag4g](https://github.com/hooto/hflag4g) commandline flags processing library for Golang
* [https://github.com/hooto/hlang4g](https://github.com/hooto/hlang4g) i18n library for golang
* [https://github.com/hooto/hcaptcha4g](https://github.com/hooto/hcaptcha4g) Captcha library for Golang


以上推荐的依赖库大多纯粹精简, Go 编程语言生态系统里包含大量优秀项目, 推荐一个项目导航清单供参考: 

* Go 第三方库导航 [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)


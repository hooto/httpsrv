# httpsrv

httpsrv 是一个轻量级、模块化、高性能的 MVC Web 框架，专为 Go 语言设计，适用于开发面向互联网的各类 API、Web 应用。

## 特性

- **模块化架构** - 业务代码通过模块组织管理，适合企业级复杂应用开发
- **轻量简洁** - 核心代码精简（约 3000 行），接口稳定可靠，便于长期维护
- **高性能** - 低内存占用，高并发稳定可靠，主流云服务商 2 核主机 QPS 可达 20000+
- **MVC 模式** - 支持标准的 MVC 架构，代码结构清晰
- **模板引擎** - 内置模板引擎，支持灵活的视图渲染
- **多语言支持** - 内置 i18n 国际化支持
- **中间件** - 支持请求过滤和拦截器
- **会话管理** - 内置会话管理功能

## 文档

- [快速开始](doc/start.md)
- [常见问题](doc/qa.md)
- [完整文档索引](doc/SUMMARY.md)

## 安装

```bash
go get -u github.com/hooto/httpsrv
```

## 快速开始

创建一个简单的 Hello World 应用：

```go
package main

import (
    "github.com/hooto/httpsrv"
)

// 定义一个控制器
type Hello struct {
    *httpsrv.Controller
}

// 定义一个 Action
func (c Hello) WorldAction() {
    c.RenderString("hello world")
}

// 创建模块
func NewModule() httpsrv.Module {
    module := httpsrv.NewModule("demo")
    module.ControllerRegister(new(Hello))
    return module
}

func main() {
    // 注册模块到全局服务
    httpsrv.GlobalService.ModuleRegister("/", NewModule())
    
    // 设置端口
    httpsrv.GlobalService.Config.HttpPort = 8080
    
    // 启动服务
    httpsrv.GlobalService.Start()
}
```

运行：

```bash
go run main.go
```

访问：

```bash
curl http://localhost:8080/hello/world/
```

输出：

```
hello world
```

## 项目结构

推荐的目录结构：

```
├─ bin/              # 编译后的可执行文件
├─ etc/              # 配置文件
├─ config/           # 配置解析代码
├─ cmd/
│  └─ server/
│     └─ main.go     # 服务入口
├─ data/             # 数据库访问层
├─ websrv/           # 模块目录
│  ├─ api-v1/        # API 模块
│  └─ frontend/      # 前端模块
│     └─ views/      # 模板文件
├─ webui/            # 静态文件
└─ var/              # 运行时数据
```

完整示例项目：[httpsrv-demo](https://github.com/hooto/httpsrv-demo)

## 推荐依赖库

httpsrv 保持核心简洁，以下是一些推荐使用的第三方库：

### 数据库
- [mysqlgo](https://github.com/lynkdb/mysqlgo) - MySQL 客户端
- [pgsqlgo](https://github.com/lynkdb/pgsqlgo) - PostgreSQL 客户端
- [redisgo](https://github.com/lynkdb/redisgo) - Redis 客户端
- [kvgo](https://github.com/lynkdb/kvgo) - 嵌入式 Key-Value 数据库

### 工具库
- [hlog4g](https://github.com/hooto/hlog4g) - 日志库
- [hini4g](https://github.com/hooto/hini4g) - INI 配置文件解析
- [hflag4g](https://github.com/hooto/hflag4g) - 命令行参数处理
- [hlang4g](https://github.com/hooto/hlang4g) - i18n 国际化
- [hcaptcha4g](https://github.com/hooto/hcaptcha4g) - 验证码生成

更多 Go 生态库可参考：[awesome-go](https://github.com/avelino/awesome-go)

## 系统要求

- **Go 版本**: 1.22 或更高
- **推荐系统**: Linux、Unix 或 macOS（Windows 未做兼容测试）

## 核心组件

- [Service](doc/service.md) - 服务容器和配置管理
- [Config](doc/config.md) - 配置文件处理
- [Module](doc/module.md) - 模块管理和路由
- [Controller](doc/controller.md) - 控制器和请求处理
- [Template](doc/template.md) - 模板渲染和视图
- [Router](doc/router.md) - 路由配置和匹配

## 扩展组件

- [log](doc/ext/log.md) - 日志记录扩展
- [data-rdb](doc/ext/data-rdb.md) - 关系数据库扩展
- [data-kv](doc/ext/data-kv.md) - Key-Value 数据库扩展
- [flag](doc/ext/flag.md) - 命令行参数扩展

## 参考项目

httpsrv 在架构设计和部分代码实现中参考过以下项目，特此感谢！

- [Revel Framework](https://github.com/revel/revel/)
- [Beego Framework](https://github.com/astaxie/beego/)

## 许可证

[Apache License 2.0](LICENSE)

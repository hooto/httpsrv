## httpsrv 简介

httpsrv 是一个轻量级、高性能，面向模块化编程的 Golang HTTP 框架，适用于开发面向互联网的各类 API、Web 等应用。

### 主要特性

* **模块化**：业务代码通过模块组织管理，企业级复杂应用开发更高效
* **轻量级**：纯粹精简（核心代码 3000+），接口稳定可靠，长周期应用的开发更容易
* **高性能**：低内存占用，高并发稳定可靠（主流云服务商的 2 核主机 QPS 性能可达 20000+）

### 适用场景

* RESTful API 服务
* Web 应用开发
* 微服务架构
* 企业级应用系统

## 文档导航

### 快速开始

* [快速开始](start.md) - 快速上手 httpsrv，了解基本用法
* [常见问题](qa.md) - 常见问题解答和故障排除

### 核心组件

* [Service](service.md) - HTTP 服务的最外层对象实例
* [Config](config.md) - HTTP 服务启动时的配置参数
* [Module](module.md) - 模块化编程的核心组件
* [Controller](controller.md) - Web MVC 中的控制器
* [Router](router.md) - HTTP 请求路由规则
* [Template](template.md) - Web MVC 中的视图模版

### 扩展组件

* [log 日志记录](ext/log.md) - 日志记录和监控
* [Model, 关系数据库](ext/data-rdb.md) - MySQL、PostgreSQL 等关系数据库集成
* [Cache, Key-Value 数据库](ext/data-kv.md) - Redis、SSDB 等 NoSQL 数据库集成
* [命令行传参](ext/flag.md) - 命令行参数处理

## 学习路径

### 初学者

1. 阅读 [简介](README.md) 了解框架特性
2. 按照 [快速开始](start.md) 创建第一个应用
3. 学习 [Controller](controller.md) 和 [Router](router.md) 掌握基本用法
4. 参考 [常见问题](qa.md) 解决遇到的问题

### 进阶开发者

1. 深入学习 [Module](module.md) 和 [Service](service.md) 了解架构设计
2. 掌握 [Template](template.md) 进行视图渲染
3. 集成扩展组件：[日志](ext/log.md)、[数据库](ext/data-rdb.md)、[缓存](ext/data-kv.md)
4. 阅读源码和示例项目

### 最佳实践

1. 使用模块化架构组织代码
2. 遵循命名规范和约定
3. 合理使用缓存提升性能
4. 配置日志和监控系统
5. 做好错误处理和异常情况处理

## 示例项目

完整的项目示例请参考：[https://github.com/hooto/httpsrv-demo](https://github.com/hooto/httpsrv-demo)

```bash
# 克隆示例项目
git clone git@github.com:hooto/httpsrv-demo.git
cd httpsrv-demo

# 运行示例
./develop-run.sh
```

## 生态系统

httpsrv 推荐的第三方依赖库：

### 数据库
* [lynkdb/mysqlgo](https://github.com/lynkdb/mysqlgo) - MySQL 客户端
* [lynkdb/pgsqlgo](https://github.com/lynkdb/pgsqlgo) - PostgreSQL 客户端
* [lynkdb/redisgo](https://github.com/lynkdb/redisgo) - Redis 客户端
* [lynkdb/ssdbgo](https://github.com/lynkdb/ssdbgo) - SSDB 客户端

### 工具库
* [hooto/hlog4g](https://github.com/hooto/hlog4g) - 日志库
* [hooto/hini4g](https://github.com/hooto/hini4g) - INI 配置文件解析
* [hooto/hflag4g](https://github.com/hooto/hflag4g) - 命令行参数处理
* [hooto/hlang4g](https://github.com/hooto/hlang4g) - 国际化支持

更多优秀的 Go 库请参考：[Go 第三方库导航](https://github.com/avelino/awesome-go)

## 技术支持

* **GitHub Issues**: [https://github.com/hooto/httpsrv/issues](https://github.com/hooto/httpsrv/issues)
* **电子邮件**: evorui at gmail dot com

## 参考

httpsrv 在架构设计和部分代码实现中参考过如下项目，特此感谢！

* Revel Framework [https://github.com/revel/revel/](https://github.com/revel/revel/)
* Beego Framework [https://github.com/astaxie/beego/](https://github.com/astaxie/beego/)

## License

Licensed under the Apache License, Version 2.0
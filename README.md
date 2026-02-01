# httpsrv

httpsrv is a lightweight, modular, high-performance MVC web framework designed for Go, suitable for developing various internet-facing APIs and web applications.

**Language:** [English](README.md) | [中文](README.zh-CN.md)

## Features

- **Modular Architecture** - Business code organized through modules, ideal for enterprise-level complex application development
- **Lightweight & Concise** - Core code is concise (~3000 lines) with stable and reliable interfaces, easy for long-term maintenance
- **High Performance** - Low memory footprint, high concurrency stability, QPS can reach 20000+ on 2-core instances from mainstream cloud providers
- **MVC Pattern** - Supports standard MVC architecture with clear code structure
- **Template Engine** - Built-in template engine supporting flexible view rendering
- **Internationalization** - Built-in i18n support for multiple languages
- **Middleware** - Supports request filters and interceptors
- **Session Management** - Built-in session management functionality

## Documentation

- [Quick Start](doc/start.md)
- [FAQ](doc/qa.md)
- [Complete Documentation Index](doc/SUMMARY.md)

## Installation

```bash
go get -u github.com/hooto/httpsrv
```

## Quick Start

Create a simple Hello World application:

```go
package main

import (
    "github.com/hooto/httpsrv"
)

// Define a controller
type Hello struct {
    *httpsrv.Controller
}

// Define an Action
func (c Hello) WorldAction() {
    c.RenderString("hello world")
}

// Create module
func NewModule() httpsrv.Module {
    module := httpsrv.NewModule("demo")
    module.ControllerRegister(new(Hello))
    return module
}

func main() {
    // Register module to global service
    httpsrv.GlobalService.ModuleRegister("/", NewModule())
    
    // Set port
    httpsrv.GlobalService.Config.HttpPort = 8080
    
    // Start service
    httpsrv.GlobalService.Start()
}
```

Run:

```bash
go run main.go
```

Visit:

```bash
curl http://localhost:8080/hello/world/
```

Output:

```
hello world
```

## Project Structure

Recommended directory structure:

```
├─ bin/              # Compiled executables
├─ etc/              # Configuration files
├─ config/           # Configuration parsing code
├─ cmd/
│  └─ server/
│     └─ main.go     # Service entry point
├─ data/             # Database access layer
├─ websrv/           # Module directory
│  ├─ api-v1/        # API module
│  └─ frontend/      # Frontend module
│     └─ views/      # Template files
├─ webui/            # Static files
└─ var/              # Runtime data
```

Complete example project: [httpsrv-demo](https://github.com/hooto/httpsrv-demo)

## Recommended Dependencies

httpsrv keeps the core concise. Here are some recommended third-party libraries:

### Database
- [mysqlgo](https://github.com/lynkdb/mysqlgo) - MySQL client
- [pgsqlgo](https://github.com/lynkdb/pgsqlgo) - PostgreSQL client
- [redisgo](https://github.com/lynkdb/redisgo) - Redis client
- [kvgo](https://github.com/lynkdb/kvgo) - Embedded Key-Value database

### Utility Libraries
- [hlog4g](https://github.com/hooto/hlog4g) - Logging library
- [hini4g](https://github.com/hooto/hini4g) - INI configuration file parsing
- [hflag4g](https://github.com/hooto/hflag4g) - Command line argument handling
- [hlang4g](https://github.com/hooto/hlang4g) - i18n internationalization
- [hcaptcha4g](https://github.com/hooto/hcaptcha4g) - CAPTCHA generation

More Go ecosystem libraries: [awesome-go](https://github.com/avelino/awesome-go)

## System Requirements

- **Go Version**: 1.22 or higher
- **Recommended Systems**: Linux, Unix, or macOS (Windows not tested for compatibility)

## Core Components

- [Service](doc/service.md) - Service container and configuration management
- [Config](doc/config.md) - Configuration file handling
- [Module](doc/module.md) - Module management and routing
- [Controller](doc/controller.md) - Controller and request handling
- [Template](doc/template.md) - Template rendering and views
- [Router](doc/router.md) - Routing configuration and matching

## Extension Components

- [log](doc/ext/log.md) - Logging extension
- [data-rdb](doc/ext/data-rdb.md) - Relational database extension
- [data-kv](doc/ext/data-kv.md) - Key-Value database extension
- [flag](doc/ext/flag.md) - Command line argument extension

## Reference Projects

httpsrv has referenced the following projects in architecture design and some code implementations. Special thanks!

- [Revel Framework](https://github.com/revel/revel/)
- [Beego Framework](https://github.com/astaxie/beego/)

## License

[Apache License 2.0](LICENSE)
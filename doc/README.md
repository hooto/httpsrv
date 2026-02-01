## httpsrv Introduction

httpsrv is a lightweight, high-performance, modular programming-oriented Golang HTTP framework, suitable for developing various internet-facing applications such as APIs and Web applications.

### Key Features

* **Modular**: Business code is organized and managed through modules, making enterprise-level complex application development more efficient
* **Lightweight**: Pure and concise (core code 3000+), stable and reliable interfaces, easier for long-term application development
* **High Performance**: Low memory footprint, stable and reliable high concurrency (2-core host QPS performance on mainstream cloud providers can reach 20000+)

### Use Cases

* RESTful API Services
* Web Application Development
* Microservices Architecture
* Enterprise Application Systems

## Documentation Navigation

### Quick Start

* [Getting Started](start.md) - Quick start with httpsrv, learn basic usage
* [FAQ](qa.md) - Common questions and troubleshooting

### Core Components

* [Service](service.md) - The outermost object instance of HTTP service
* [Config](config.md) - Configuration parameters for HTTP service startup
* [Module](module.md) - Core component for modular programming
* [Controller](controller.md) - Controller in Web MVC
* [Router](router.md) - HTTP request routing rules
* [Template](template.md) - View template in Web MVC

### Extension Components

* [log](ext/log.md) - Logging and monitoring
* [Model, Relational Database](ext/data-rdb.md) - MySQL, PostgreSQL and other relational database integration
* [Cache, Key-Value Database](ext/data-kv.md) - Redis, SSDB and other NoSQL database integration
* [Command Line Arguments](ext/flag.md) - Command line parameter processing

## Learning Path

### Beginners

1. Read [Introduction](README.md) to understand framework features
2. Follow [Getting Started](start.md) to create your first application
3. Learn [Controller](controller.md) and [Router](router.md) to master basic usage
4. Refer to [FAQ](qa.md) to solve problems you encounter

### Advanced Developers

1. Deep dive into [Module](module.md) and [Service](service.md) to understand architecture design
2. Master [Template](template.md) for view rendering
3. Integrate extension components: [logging](ext/log.md), [database](ext/data-rdb.md), [cache](ext/data-kv.md)
4. Read source code and example projects

### Best Practices

1. Use modular architecture to organize code
2. Follow naming conventions and agreements
3. Use caching appropriately to improve performance
4. Configure logging and monitoring systems
5. Handle errors and exceptional situations properly

## Example Project

For complete project examples, please refer to: [https://github.com/hooto/httpsrv-demo](https://github.com/hooto/httpsrv-demo)

```bash
# Clone example project
git clone git@github.com:hooto/httpsrv-demo.git
cd httpsrv-demo

# Run example
./develop-run.sh
```

## Ecosystem

Third-party libraries recommended by httpsrv:

### Database
* [lynkdb/mysqlgo](https://github.com/lynkdb/mysqlgo) - MySQL client
* [lynkdb/pgsqlgo](https://github.com/lynkdb/pgsqlgo) - PostgreSQL client
* [lynkdb/redisgo](https://github.com/lynkdb/redisgo) - Redis client
* [lynkdb/ssdbgo](https://github.com/lynkdb/ssdbgo) - SSDB client

### Utilities
* [hooto/hlog4g](https://github.com/hooto/hlog4g) - Logging library
* [hooto/hini4g](https://github.com/hooto/hini4g) - INI config file parsing
* [hooto/hflag4g](https://github.com/hooto/hflag4g) - Command line parameter processing
* [hooto/hlang4g](https://github.com/hooto/hlang4g) - Internationalization support

For more excellent Go libraries, please refer to: [Go Third-party Libraries Directory](https://github.com/avelino/awesome-go)

## Technical Support

* **GitHub Issues**: [https://github.com/hooto/httpsrv/issues](https://github.com/hooto/httpsrv/issues)
* **Email**: evorui at gmail dot com

## References

httpsrv refers to the following projects in its architecture design and some code implementations. Thank you!

* Revel Framework [https://github.com/revel/revel/](https://github.com/revel/revel/)
* Beego Framework [https://github.com/astaxie/beego/](https://github.com/astaxie/beego/)

## License

Licensed under the Apache License, Version 2.0
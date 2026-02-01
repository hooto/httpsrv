## System Environment Preparation

httpsrv is a Golang HTTP framework. Please ensure that the Go development environment is properly installed and configured. For details, please refer to the official website:

* Official Website [https://golang.org](https://golang.org)
* Chinese Mirror [https://golang.google.cn](https://golang.google.cn)

> Note: It is recommended to develop and deploy httpsrv-based applications on Linux, Unix, or macOS systems. Windows OS is not common in enterprise applications, so no compatibility testing has been done.

Install or update httpsrv:

``` shell
go get -u github.com/hooto/httpsrv
```

If the above steps are successful, let's start with a simple example.

## Start a "Hello World" Example

Create a file main.go and edit it:

``` go
package main

import (
	"github.com/hooto/httpsrv"
)

// Create a new Controller
type Hello struct {
	*httpsrv.Controller
}

func (c Hello) WorldAction() {
	c.RenderString("hello world")
}

// Build a module example
func NewModule() httpsrv.Module {

	// Initialize an empty module
	module := httpsrv.NewModule("demo")
    
	// Register a controller to the module
	module.ControllerRegister(new(Hello))

	return module
}

// Global entry point
func main() {

	// Register the module to the service container
	httpsrv.GlobalService.ModuleRegister("/", NewModule())

	// Set service port
	httpsrv.GlobalService.Config.HttpPort = 8080

	// Start service
	httpsrv.GlobalService.Start()
}
```

Run this example:

``` shell
go run main.go
```

You can access the newly created service via http://localhost:8080/hello/world/

## Start a Formal Project

The "hello world" above is just a simple example. Actual projects will be more complex:

### Recommended Directory Structure

As an HTTP framework oriented towards modular programming, it is recommended to create independent file directories for each module's business logic to organize and manage code, making the overall project structure clear and easy to manage, as shown in the example.

``` shell
├─ bin/
│  └─ cmd-server // Compiled executable file: go build -o bin/cmd-server cmd/server/main.go
├─ etc/
│  └─ config.ini,json,yaml, ... // Configuration file storage
├─ config/
│  └─ config.go // Configuration file parsing
├─ cmd/
│  └─ server/
│     └─ main.go // Server service entry point
├─ data/
│  └─ data.go // Database, storage access (MySQL, PostgreSQL, Redis, ...)
├─ websrv/ // Modules
│  ├─ api-v1/ // Module api-v1
│  │  ├─ setup.go
│  │  └─ controller-a.go
│  └─ frontend/ // Module frontend
│     ├─ setup.go
│     ├─ controller-a.go
│     └─ views/ // HTML view template files
│        └─ controller-a/
│           ├─ action-a.tpl
│           └─ action-b.tpl
├─ webui/ // Static file modules
│  └─ a/ // Static file module a
│     ├─ img/*
│     ├─ css/*
│     └─ js/*
└─ var/
```

> Note: Golang has specific import path rules, so please ensure the directory structure is correct.

For a complete code example with the above directory structure, please refer to [https://github.com/hooto/httpsrv-demo](https://github.com/hooto/httpsrv-demo). You can clone it with git and run it:

``` shell
# Use git to clone the example code
git clone git@github.com:hooto/httpsrv-demo.git
cd httpsrv-demo

# Run the example
./develop-run.sh 
I 2019-07-13 17:21:37.643316 config.go:29] project prefix path /opt/gopath/src/github.com/hooto/httpsrv-demo
I 2019-07-13 17:21:37.644743 service.go:240] lessgo/httpsrv: listening on tcp/0.0.0.0:8080
```

You can access the service in the above example via http://localhost:8080/. If you are starting a brand new project, this example code can be used as a template.

### Business Logic Dependencies

httpsrv is a purely concise HTTP framework that only encapsulates business common interfaces related to HTTP request/response. For business middleware such as Model, ORM, Cache, etc. involved in Web development, it does not provide built-in support, but relies on third-party libraries as needed.

* Go client for MySQL [https://github.com/lynkdb/mysqlgo](https://github.com/lynkdb/mysqlgo)
* Go client for PostgreSQL [https://github.com/lynkdb/pgsqlgo](https://github.com/lynkdb/pgsqlgo)
* Go client for Redis [https://github.com/lynkdb/redisgo](https://github.com/lynkdb/redisgo)
* Go client for SSDB [https://github.com/lynkdb/ssdbgo](https://github.com/lynkdb/ssdbgo)
* An embedded Key-Value database library for Go language [https://github.com/lynkdb/kvgo](https://github.com/lynkdb/kvgo)
* Log library for Golang [https://github.com/hooto/hlog4g](https://github.com/hooto/hlog4g)
* INI file read library for Golang [https://github.com/hooto/hini4g](https://github.com/hooto/hini4g)
* Commandline flags processing library for Golang [https://github.com/hooto/hflag4g](https://github.com/hooto/hflag4g)
* i18n library for golang [https://github.com/hooto/hlang4g](https://github.com/hooto/hlang4g)
* Captcha library for Golang [https://github.com/hooto/hcaptcha4g](https://github.com/hooto/hcaptcha4g)

Most of the recommended dependencies above are purely concise. The Go programming language ecosystem contains a large number of excellent projects. Here is a recommended project navigation list for reference:

* Go Third-party Libraries Directory [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)
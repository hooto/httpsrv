## httpsrv

httpsrv is a Lightweight, Modular, High Performance MVC web framework for the Go language.

Documents:
* 中文文档 [https://www.hooto.com/gdoc/view/hooto-httpsrv/](https://www.hooto.com/gdoc/view/hooto-httpsrv/)

## Quick Start

Install httpsrv framework

```shell
go get -u github.com/hooto/httpsrv

```

first hello world demo

```go
package main

import (
    "github.com/hooto/httpsrv"
)

type Index struct {
    *httpsrv.Controller
}

func (c Index) IndexAction() {
    c.RenderString("hello world")
}

// init one module
func NewModule() httpsrv.Module {

	module := httpsrv.NewModule("default")

	//register controller to module
	module.ControllerRegister(new(Index))

	return module
}


func main() {

    // register module to httpsrv
    httpsrv.GlobalService.ModuleRegister("/", NewModule)

    // listening on port 8080
    httpsrv.GlobalService.Config.HttpPort = 8080

    // start
    httpsrv.GlobalService.Start()
}
```

## Licensing
Licensed under the Apache License, Version 2.0


## httpsrv

httpsrv is a Lightweight, Modular, High Performance MVC web framework for the Go language.

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

func main() {

    // init one module
    module := httpsrv.NewModule("default")
    
    // register controller to module
    module.ControllerRegister(new(Index))

    // register module to httpsrv
    httpsrv.GlobalService.ModuleRegister("/", module)

    // listening on port 18080
    httpsrv.GlobalService.Config.HttpPort = 18080

    // start
    httpsrv.GlobalService.Start()
}
```

## Licensing
Licensed under the Apache License, Version 2.0


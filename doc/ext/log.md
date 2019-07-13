## log

httpsrv 没有内置的 log 接口，如果有这方面需求, 可使用如下依赖库:

* hooto/hlog4g [https://github.com/hooto/hlog4g](https://github.com/hooto/hlog4g) 

安装:

``` shell
go get -u github.com/hooto/hlog4g/hlog
```

使用:

``` go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {

	// API:Print
	hlog.Print("info", "started")
	hlog.Print("error", "the error code/message: ", 400, "/", "bad request")

	// API::Printf
	hlog.Printf("error", "the error code/message: %d/%s", 400, "bad request")

	select {}
}
```

日志输出格式样式:
``` shell
./main --logtostderr=true

I 2019-07-07 20:39:16.448449 main.go:10] started
E 2019-07-07 20:39:16.448476 main.go:11] the error code/message: 400/bad request
E 2019-07-07 20:39:16.448494 main.go:14] the error code/message: 400/bad request
```

注:

* 调试开发时，如果需要打印日志到当前命令终端，可在命令后加入 `--logtostderr=true`
* 正式部署时，如果需要将日志输出到本地文件，可以在可执行命令后加入 `--log_dir=/path/of/log/`


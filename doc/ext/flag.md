## flag

httpsrv 没有内置的终端命令行参数处理接口，如果有这方面需求, 可使用如下依赖库:

* [https://github.com/hooto/hflag4g](https://github.com/hooto/hflag4g) 

使用示例:

``` go
import (
	"fmt"

	"github.com/hooto/hflag4g/hflag"
)

func main() {
	fmt.Println("flag value:", hflag.Value("server_name").String())
}
```

执行

``` shell
go build -o bin/demo-server main.go
./bin/demo-server --server_name=cms
```

输出

``` shell
flag value: cms
```


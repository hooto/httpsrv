## log

httpsrv 没有内置的 log 接口，如果有这方面需求，可使用如下依赖库：

* hooto/hlog4g [https://github.com/hooto/hlog4g](https://github.com/hooto/hlog4g)

## 安装 hlog4g

```bash
go get -u github.com/hooto/hlog4g/hlog
```

## 基本使用

### Print 和 Printf

```go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {
	// Print: 可变参数打印
	hlog.Print("info", "started")
	hlog.Print("error", "the error code/message: ", 400, "/", "bad request")

	// Printf: 格式化打印
	hlog.Printf("error", "the error code/message: %d/%s", 400, "bad request")

	select {}
}
```

### 日志级别

hlog4g 支持多个日志级别：

```go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {
	// DEBUG 级别
	hlog.Debug("debug message")

	// INFO 级别（默认）
	hlog.Info("service started")

	// WARNING 级别
	hlog.Warning("this is a warning")

	// ERROR 级别
	hlog.Error("an error occurred")

	// FATAL 级别（会终止程序）
	// hlog.Fatal("fatal error")
}
```

### 日志输出格式

```shell
./main --logtostderr=true

I 2019-07-07 20:39:16.448449 main.go:10] started
E 2019-07-07 20:39:16.448476 main.go:11] the error code/message: 400/bad request
E 2019-07-07 20:39:16.448494 main.go:14] the error code/message: 400/bad request
```

日志格式说明：
- `I` / `E` / `W` / `D` / `F`: 日志级别
- `2019-07-07 20:39:16.448449`: 时间戳（精确到微秒）
- `main.go:10`: 文件名和行号
- `started`: 日志内容

## 命令行参数配置

hlog4g 支持通过命令行参数配置日志输出：

### 输出到终端

```bash
# 将日志输出到标准错误输出
./main --logtostderr=true
```

### 输出到文件

```bash
# 将日志输出到指定目录
./main --log_dir=/var/log/myapp

# 同时输出到终端和文件
./main --logtostderr=true --log_dir=/var/log/myapp
```

### 设置日志级别

```bash
# 只输出 INFO 级别及以上的日志
./main --v=0

# 输出 WARNING 级别及以上的日志
./main --v=-1

# 输出 DEBUG 级别及以上的日志
./main --v=1
```

### 设置日志文件大小限制

```bash
# 单个日志文件最大 100MB
./main --logtostderr=false --log_dir=/var/log/myapp --log_max_size=100

# 最多保留 10 个日志文件
./main --log_dir=/var/log/myapp --log_max_files=10

# 日志文件保留天数（默认 7 天）
./main --log_dir=/var/log/myapp --log_max_age=30
```

## 在 Controller 中使用日志

```go
package controller

import (
	"github.com/hooto/httpsrv"
	"github.com/hooto/hlog4g/hlog"
)

type User struct {
	*httpsrv.Controller
}

func (c User) LoginAction() {
	username := c.Params.Value("username")
	password := c.Params.Value("password")

	hlog.Info("user login attempt", "username:", username)

	// 验证用户
	if username == "admin" && password == "password" {
		hlog.Info("user login success", "username:", username)
		
		c.Session.Set("user_id", "1")
		c.Session.Set("username", username)
		
		c.RenderJson(map[string]string{
			"status": "success",
			"message": "登录成功",
		})
	} else {
		hlog.Warning("user login failed", "username:", username, "reason: invalid credentials")
		c.RenderError(401, "用户名或密码错误")
	}
}

func (c User) RegisterAction() {
	username := c.Params.Value("username")
	email := c.Params.Value("email")
	password := c.Params.Value("password")

	hlog.Info("user register attempt", "username:", username, "email:", email)

	// 检查用户是否已存在
	if userExists(username) {
		hlog.Warning("user register failed", "username:", username, "reason: user already exists")
		c.RenderError(400, "用户名已存在")
		return
	}

	// 创建用户
	if err := createUser(username, email, password); err != nil {
		hlog.Error("user register failed", "username:", username, "error:", err.Error())
		c.RenderError(500, "注册失败")
		return
	}

	hlog.Info("user register success", "username:", username)
	c.RenderJson(map[string]string{
		"status": "success",
		"message": "注册成功",
	})
}

func (c User) DeleteAction() {
	id := c.Params.IntValue("id")

	hlog.Info("user delete attempt", "user_id:", id)

	if err := deleteUser(id); err != nil {
		hlog.Error("user delete failed", "user_id:", id, "error:", err.Error())
		c.RenderError(500, "删除失败")
		return
	}

	hlog.Info("user delete success", "user_id:", id)
	c.RenderJson(map[string]string{
		"status": "success",
		"message": "删除成功",
	})
}
```

## 结构化日志

hlog4g 支持结构化日志，可以记录键值对格式的日志：

```go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {
	// 记录结构化日志
	hlog.Info("user_action", 
		"action", "login",
		"user_id", "12345",
		"ip", "192.168.1.1",
		"status", "success",
		"duration_ms", 45,
	)

	// 记录错误信息
	hlog.Error("database_error",
		"operation", "query",
		"table", "users",
		"error", "connection timeout",
		"retry_count", 3,
	)
}
```

## 性能日志

记录 API 请求的性能指标：

```go
package middleware

import (
	"time"
	"github.com/hooto/hlog4g/hlog"
	"github.com/hooto/httpsrv"
)

func LoggingFilter(c *httpsrv.Controller, fc []httpsrv.Filter) {
	startTime := time.Now()

	// 记录请求开始
	hlog.Info("request_start",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"remote_addr", c.Request.RemoteAddr,
	)

	// 继续执行后续 Filter
	fc[0](c, fc[1:])

	// 记录请求结束
	duration := time.Since(startTime)
	hlog.Info("request_end",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status_code", c.Response.Status,
		"duration_ms", duration.Milliseconds(),
	)
}
```

在 Service 中注册 Filter：

```go
func main() {
	// 注册日志 Filter
	httpsrv.GlobalService.Filters = []httpsrv.Filter{
		middleware.LoggingFilter,
	}

	// 注册模块
	httpsrv.GlobalService.HandleModule("/", NewModule())
	httpsrv.GlobalService.Start()
}
```

## 错误堆栈日志

记录详细的错误堆栈信息：

```go
package controller

import (
	"github.com/hooto/httpsrv"
	"github.com/hooto/hlog4g/hlog"
)

type Task struct {
	*httpsrv.Controller
}

func (c Task) ExecuteAction() {
	taskID := c.Params.Value("id")

	hlog.Info("task execution start", "task_id:", taskID)

	if err := executeTask(taskID); err != nil {
		// 记录错误堆栈
		hlog.Error("task execution failed", 
			"task_id:", taskID,
			"error:", err.Error(),
			"stack", hlog.Stack(),
		)
		c.RenderError(500, "任务执行失败")
		return
	}

	hlog.Info("task execution success", "task_id:", taskID)
	c.RenderJson(map[string]string{
		"status": "success",
		"message": "任务执行成功",
	})
}
```

## 日志轮转配置

### 程序内配置

```go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {
	// 配置日志输出
	hlog.SetOutput("file", "./logs/app.log")
	
	// 配置日志级别
	hlog.SetLevel(hlog.LevelInfo)
	
	// 配置日志格式
	hlog.SetFormat(hlog.FormatText)
	
	// 配置日志轮转
	hlog.SetRotation(&hlog.RotationConfig{
		MaxSize:    100,  // 单个文件最大 100MB
		MaxFiles:   10,   // 最多保留 10 个文件
		MaxAge:     7,    // 保留 7 天
		Compress:   true, // 压缩旧日志文件
	})

	// 使用日志
	hlog.Info("application started")
}
```

### 配置文件示例

创建 `logging.conf` 文件：

```ini
[log]
# 日志级别: 0=DEBUG, 1=INFO, 2=WARNING, 3=ERROR, 4=FATAL
level = 1

# 日志输出方式: stdout, file, both
output = both

# 日志文件路径
log_dir = ./logs

# 日志文件名前缀
log_file_prefix = app

# 单个日志文件最大大小 (MB)
log_max_size = 100

# 最多保留的日志文件数
log_max_files = 10

# 日志文件保留天数
log_max_age = 7

# 是否压缩旧日志文件
log_compress = true

# 日志格式: text, json
log_format = text

# 是否包含调用堆栈
log_stack = false
```

在程序中加载配置：

```go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {
	// 从配置文件加载
	if err := hlog.LoadConfig("logging.conf"); err != nil {
		panic(err)
	}

	// 使用日志
	hlog.Info("application started with config")
}
```

## 生产环境最佳实践

### 1. 分离不同级别的日志

```go
package config

import (
	"github.com/hooto/hlog4g/hlog"
)

func SetupLogging() {
	// 设置 INFO 级别及以上日志输出到文件
	hlog.SetOutputByLevel(hlog.LevelInfo, "file", "./logs/info.log")

	// 设置 ERROR 级别及以上日志输出到单独文件
	hlog.SetOutputByLevel(hlog.LevelError, "file", "./logs/error.log")

	// FATAL 级别同时输出到终端和文件
	hlog.SetOutputByLevel(hlog.LevelFatal, "both", "./logs/fatal.log")
}
```

### 2. 敏感信息脱敏

```go
package util

import (
	"strings"
)

// 脱敏处理
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***.***"
	}
	username := parts[0]
	if len(username) > 3 {
		username = username[:2] + "***"
	}
	return username + "@" + parts[1]
}

func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return "***"
	}
	return phone[:3] + "****" + phone[7:]
}

func MaskIDCard(idCard string) string {
	if len(idCard) < 10 {
		return "***"
	}
	return idCard[:6] + "********" + idCard[len(idCard)-4:]
}
```

在日志中使用：

```go
hlog.Info("user_login", 
	"username", username,
	"email", util.MaskEmail(email),
	"phone", util.MaskPhone(phone),
	"ip", c.Request.RemoteAddr,
)
```

### 3. 异步日志

对于高性能场景，可以使用异步日志：

```go
package main

import (
	"github.com/hooto/hlog4g/hlog"
)

func main() {
	// 启用异步日志
	hlog.SetAsync(true, 1000) // 缓冲区大小 1000

	// 使用日志
	for i := 0; i < 10000; i++ {
		hlog.Info("async_log", "index", i)
	}

	// 确保所有日志都写入
	hlog.Flush()
}
```

### 4. 日志上下文追踪

为每个请求添加追踪 ID：

```go
package middleware

import (
	"github.com/google/uuid"
	"github.com/hooto/httpsrv"
	"github.com/hooto/hlog4g/hlog"
)

func TraceFilter(c *httpsrv.Controller, fc []httpsrv.Filter) {
	// 生成或获取追踪 ID
	traceID := c.Request.Header.Get("X-Trace-ID")
	if traceID == "" {
		traceID = uuid.New().String()
	}

	// 设置到响应头
	c.Response.Out.Header().Set("X-Trace-ID", traceID)

	// 添加到日志上下文
	hlog.SetContext("trace_id", traceID)

	// 继续执行
	fc[0](c, fc[1:])

	// 清除上下文
	hlog.ClearContext()
}
```

在 Controller 中使用：

```go
func (c User) LoginAction() {
	// 日志会自动包含 trace_id
	hlog.Info("user_login", "username:", username)
}
```

### 5. 日志监控和告警

```go
package monitor

import (
	"time"
	"github.com/hooto/hlog4g/hlog"
)

type ErrorCount struct {
	count     int
	startTime time.Time
}

var errorCounter = make(map[string]*ErrorCount)

func CheckErrors() {
	for key, counter := range errorCounter {
		// 1分钟内错误超过 10 次，发送告警
		if counter.count > 10 && time.Since(counter.startTime) < time.Minute {
			hlog.Error("high_error_rate",
				"error_type", key,
				"count", counter.count,
				"duration", time.Since(counter.startTime),
			)
			
			// 发送告警通知（邮件、短信等）
			sendAlert(key, counter.count)
		}
	}
}

func RecordError(errorType string) {
	if _, ok := errorCounter[errorType]; !ok {
		errorCounter[errorType] = &ErrorCount{
			startTime: time.Now(),
		}
	}
	errorCounter[errorType].count++
}
```

## 注意事项

1. **性能考虑**：日志操作会带来一定的性能开销，生产环境建议使用异步日志
2. **磁盘空间**：合理配置日志轮转策略，避免日志文件占用过多磁盘空间
3. **敏感信息**：记录日志时注意不要包含密码、密钥等敏感信息
4. **日志级别**：生产环境建议使用 INFO 或 WARNING 级别，开发环境使用 DEBUG 级别
5. **日志格式**：建议使用结构化日志，便于后续分析和检索

注：

* 调试开发时，如果需要打印日志到当前命令终端，可在命令后加入 `--logtostderr=true`
* 正式部署时，如果需要将日志输出到本地文件，可以在可执行命令后加入 `--log_dir=/path/of/log/`
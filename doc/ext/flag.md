## Command Line Arguments

httpsrv does not have built-in command line argument handling. If you need this functionality, you can use the following dependency library:

* [https://github.com/hooto/hflag4g](https://github.com/hooto/hflag4g)

## Basic Usage

```go
import (
	"fmt"

	"github.com/hooto/hflag4g/hflag"
)

func main() {
	fmt.Println("flag value:", hflag.Value("server_name").String())
}
```

Execute:

```bash
go build -o bin/demo-server main.go
./bin/demo-server --server_name=cms
```

Output:

```bash
flag value: cms
```

## Common Use Cases

### 1. Define Flags

```go
package main

import (
	"fmt"
	"github.com/hooto/hflag4g/hflag"
)

func init() {
	// Register flags
	hflag.RegisterString("server_name", "demo", "Server name")
	hflag.RegisterInt("port", 8080, "Server port")
	hflag.RegisterBool("verbose", false, "Enable verbose output")
}

func main() {
	// Parse command line arguments
	hflag.Parse()

	// Get flag values
	serverName := hflag.Value("server_name").String()
	port := hflag.Value("port").Int()
	verbose := hflag.Value("verbose").Bool()

	fmt.Printf("Server: %s, Port: %d, Verbose: %v\n", serverName, port, verbose)
}
```

Execute:

```bash
./demo-server --server_name=api --port=9000 --verbose=true
```

### 2. Environment Variable Support

```go
package main

import (
	"fmt"
	"os"
	"github.com/hooto/hflag4g/hflag"
)

func init() {
	// Register flags with environment variable support
	hflag.RegisterStringEnv("db_host", "localhost", "Database host", "DB_HOST")
	hflag.RegisterIntEnv("db_port", 3306, "Database port", "DB_PORT")
}

func main() {
	hflag.Parse()

	dbHost := hflag.Value("db_host").String()
	dbPort := hflag.Value("db_port").Int()

	fmt.Printf("Database: %s:%d\n", dbHost, dbPort)
}
```

Execute:

```bash
# Set environment variables first
export DB_HOST=prod.db.example.com
export DB_PORT=5432

./demo-server
# Output: Database: prod.db.example.com:5432
```

### 3. Required Flags

```go
func init() {
	// Mark flag as required
	hflag.RegisterString("api_key", "", "API key").Required(true)
}

func main() {
	if err := hflag.Parse(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	apiKey := hflag.Value("api_key").String()
	fmt.Println("API Key:", apiKey)
}
```

### 4. Flag Groups and Help

```go
func init() {
	// Server configuration
	hflag.RegisterStringGroup("server", "host", "0.0.0.0", "Server host address")
	hflag.RegisterIntGroup("server", "port", 8080, "Server port")

	// Database configuration
	hflag.RegisterStringGroup("database", "host", "localhost", "Database host")
	hflag.RegisterIntGroup("database", "port", 3306, "Database port")
	hflag.RegisterStringGroup("database", "name", "myapp", "Database name")
}

func main() {
	// Show help if -h or --help is provided
	if hflag.Bool("help") {
		hflag.PrintHelp()
		return
	}

	hflag.Parse()

	// Get values
	serverHost := hflag.Value("server.host").String()
	serverPort := hflag.Value("server.port").Int()

	dbHost := hflag.Value("database.host").String()
	dbPort := hflag.Value("database.port").Int()
	dbName := hflag.Value("database.name").String()

	fmt.Printf("Server: %s:%d\n", serverHost, serverPort)
	fmt.Printf("Database: %s:%d/%s\n", dbHost, dbPort, dbName)
}
```

Execute:

```bash
./demo-server --help
```

### 5. Integration with httpsrv

```go
package main

import (
	"github.com/hooto/httpsrv"
	"github.com/hooto/hflag4g/hflag"
)

func init() {
	hflag.RegisterString("config", "config.ini", "Configuration file path")
	hflag.RegisterInt("port", 8080, "Server port")
	hflag.RegisterBool("debug", false, "Enable debug mode")
}

func main() {
	hflag.Parse()

	// Apply flags to httpsrv configuration
	configFile := hflag.Value("config").String()
	port := hflag.Value("port").Int()
	debug := hflag.Value("debug").Bool()

	// Load configuration from file
	if err := LoadConfig(configFile); err != nil {
		panic(err)
	}

	// Override with command line arguments
	httpsrv.GlobalService.Config.HttpPort = uint16(port)

	// Set debug mode
	if debug {
		httpsrv.GlobalService.Filters = append(
			httpsrv.GlobalService.Filters,
			DebugFilter,
		)
	}

	// Register modules
	httpsrv.GlobalService.HandleModule("/", NewModule())

	// Start service
	httpsrv.GlobalService.Start()
}
```

Execute:

```bash
./demo-server --config=/path/to/prod-config.ini --port=9000 --debug=true
```

### 6. Boolean Flags

```go
func init() {
	hflag.RegisterBool("verbose", false, "Enable verbose logging")
	hflag.RegisterBool("dev", false, "Development mode")
}

func main() {
	hflag.Parse()

	verbose := hflag.Value("verbose").Bool()
	dev := hflag.Value("dev").Bool()

	if verbose {
		fmt.Println("Verbose mode enabled")
	}

	if dev {
		fmt.Println("Development mode enabled")
	}
}
```

Execute:

```bash
./demo-server --verbose --dev
```

### 7. Slice Flags

```go
func init() {
	hflag.RegisterStringSlice("allowed_ips", []string{}, "Allowed IP addresses")
	hflag.RegisterIntSlice("ports", []int{8080, 8081}, "Allowed ports")
}

func main() {
	hflag.Parse()

	allowedIPs := hflag.Value("allowed_ips").StringSlice()
	ports := hflag.Value("ports").IntSlice()

	fmt.Printf("Allowed IPs: %v\n", allowedIPs)
	fmt.Printf("Allowed Ports: %v\n", ports)
}
```

Execute:

```bash
./demo-server --allowed_ips=192.168.1.1,192.168.1.2 --ports=3000,3001,3002
```

### 8. Duration Flags

```go
func init() {
	hflag.RegisterDuration("timeout", 30*time.Second, "Request timeout")
	hflag.RegisterDuration("interval", 5*time.Minute, "Health check interval")
}

func main() {
	hflag.Parse()

	timeout := hflag.Value("timeout").Duration()
	interval := hflag.Value("interval").Duration()

	fmt.Printf("Timeout: %v\n", timeout)
	fmt.Printf("Interval: %v\n", interval)
}
```

Execute:

```bash
./demo-server --timeout=1m30s --interval=10m
```

### 9. Custom Flag Types

```go
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarning
	LogError
)

func (l *LogLevel) String() string {
	return [...]string{"debug", "info", "warning", "error"}[*l]
}

func (l *LogLevel) Set(value string) error {
	switch strings.ToLower(value) {
	case "debug":
		*l = LogDebug
	case "info":
		*l = LogInfo
	case "warning":
		*l = LogWarning
	case "error":
		*l = LogError
	default:
		return fmt.Errorf("invalid log level: %s", value)
	}
	return nil
}

func init() {
	var logLevel LogLevel
	hflag.RegisterVar("log_level", &logLevel, LogInfo, "Log level (debug/info/warning/error)")
}

func main() {
	hflag.Parse()
	// Use log level...
}
```

Execute:

```bash
./demo-server --log_level=warning
```

## Best Practices

### 1. Flag Naming Convention

```go
// Use lowercase with underscores
hflag.RegisterString("server_host", "localhost", "Server host")  // Good
hflag.RegisterString("serverhost", "localhost", "Server host")   // Less readable

// Use descriptive names
hflag.RegisterInt("http_port", 8080, "HTTP server port")     // Good
hflag.RegisterInt("port", 8080, "Port")                      // Not clear enough
```

### 2. Provide Default Values

```go
// Always provide sensible defaults
hflag.RegisterString("db_host", "localhost", "Database host")
hflag.RegisterInt("db_port", 3306, "Database port")
hflag.RegisterString("log_level", "info", "Log level")

// For required flags, set empty default and mark as required
hflag.RegisterString("api_key", "", "API key").Required(true)
```

### 3. Add Help Text

```go
// Clear, concise help text
hflag.RegisterString("config", "", "Path to configuration file")
hflag.RegisterInt("port", 8080, "Port number to listen on")
hflag.RegisterBool("verbose", false, "Enable verbose logging output")
```

### 4. Configuration File + Flags

```go
func main() {
	hflag.Parse()

	// Load config file first
	configFile := hflag.Value("config").String()
	if configFile != "" {
		LoadConfigFile(configFile)
	}

	// Override with command line flags
	if hflag.IsSet("port") {
		Config.Port = hflag.Value("port").Int()
	}

	if hflag.IsSet("verbose") {
		Config.Verbose = hflag.Value("verbose").Bool()
	}
}
```

Execute:

```bash
./demo-server --config=prod.ini --port=9000 --verbose
```

### 5. Validation

```go
func main() {
	hflag.Parse()

	port := hflag.Value("port").Int()

	// Validate port range
	if port < 1024 || port > 65535 {
		fmt.Println("Error: Port must be between 1024 and 65535")
		os.Exit(1)
	}

	// Validate API key
	apiKey := hflag.Value("api_key").String()
	if len(apiKey) < 32 {
		fmt.Println("Error: API key must be at least 32 characters")
		os.Exit(1)
	}
}
```

## Other Recommendations

The Go programming language ecosystem contains a large number of excellent projects. Here is a recommended project navigation list for reference:

* Go Third-party Libraries Directory [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)

### Other Common Go Flag Libraries

* Standard library flag [https://pkg.go.dev/flag](https://pkg.go.dev/flag) - Go's built-in flag package
* cobra [https://github.com/spf13/cobra](https://github.com/spf13/cobra) - A popular CLI application framework
* kingpin [https://github.com/alecthomas/kingpin](https://github.com/alecthomas/kingpin) - A more powerful flag parser
# tlog

A reusable Go logging package built on [Zap](https://github.com/uber-go/zap), with integrations for **Gin** and **GORM**.

## Features

- **Fast & Structured**: Built on Zap for high-performance structured logging
- **Multi-output**: Console and file output with rotation (via [lumberjack](https://github.com/natefinch/lumberjack))
- **Environment-aware**: Development (colored console) and production (JSON) modes
- **Context-aware**: Request tracing with `request_id`, `user_id`, `trace_id`
- **Gin Middleware**: Request logging with body capture on errors
- **GORM Adapter**: SQL logging with slow query detection
- **Vietnam Timezone**: Default timezone set to UTC+7

## Installation

```bash
go get github.com/thienel/tlog
```

---

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/thienel/tlog"
    "go.uber.org/zap"
)

func main() {
    // Initialize with defaults (development mode, console output, Vietnam timezone)
    if err := tlog.InitWithDefaults(); err != nil {
        panic(err)
    }
    defer tlog.Sync()

    // Log messages with different levels
    tlog.Debug("Debug message", zap.String("key", "value"))
    tlog.Info("Application started", zap.Int("port", 8080))
    tlog.Warn("Warning message", zap.String("warning", "low disk space"))
    tlog.Error("Error occurred", zap.Error(errors.New("something went wrong")))
    
    // Fatal will log and exit(1)
    // tlog.Fatal("Fatal error", zap.Error(err))
    
    // Panic will log and panic
    // tlog.Panic("Panic error", zap.Error(err))
}
```

### Using Logger Instance

```go
// Get global logger
logger := tlog.L()
logger.Info("Using logger instance")

// Get sugared logger (slower but more convenient)
sugar := tlog.S()
sugar.Infof("Hello %s", "world")
sugar.Infow("User login", "user_id", 123, "ip", "192.168.1.1")

// Create child logger with fields
childLogger := tlog.With(zap.String("service", "user-service"))
childLogger.Info("This log will always have service field")
```

---

## Configuration

### Config Struct

```go
type Config struct {
    Environment   string          // "development" | "production"
    Level         string          // "debug" | "info" | "warn" | "error"
    AppName       string          // Service identifier in logs
    Version       string          // Application version in logs
    
    // Console output
    EnableConsole bool            // Enable stdout output
    
    // File output
    EnableFile    bool            // Enable file output
    FilePath      string          // Path to log file
    MaxSizeMB     int             // Maximum size before rotation (MB)
    MaxBackups    int             // Number of old files to keep
    MaxAgeDays    int             // Maximum age in days
    Compress      bool            // Compress rotated files
    
    Timezone      *time.Location  // Timezone for timestamps
}
```

### Default Values

| Option | Default | Description |
|--------|---------|-------------|
| `Environment` | `"development"` | Log format (`development`=colored console, `production`=JSON) |
| `Level` | `"info"` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `AppName` | `"app"` | Service identifier in logs (`service` field) |
| `Version` | `"1.0.0"` | Application version in logs (`version` field) |
| `EnableConsole` | `true` | Enable stdout output |
| `EnableFile` | `false` | Enable file output |
| `FilePath` | `"logs/app.log"` | Log file path |
| `MaxSizeMB` | `100` | Max file size before rotation |
| `MaxBackups` | `3` | Number of old files to keep (0=unlimited) |
| `MaxAgeDays` | `30` | Max days to keep files (0=unlimited) |
| `Compress` | `true` | Compress rotated files with gzip |
| `Timezone` | `Asia/Ho_Chi_Minh` | Timezone for timestamps (UTC+7) |

### Configuration Examples

#### Development Mode (Default)

```go
// Using InitWithDefaults - recommended for development
tlog.InitWithDefaults()

// Or explicitly
cfg := tlog.DefaultConfig()
tlog.Init(cfg)
```

Output:
```
15:04:05    INFO    tlog/main.go:10    Application started    {"port": 8080}
```

#### Production Mode (JSON)

```go
cfg := tlog.DefaultConfig().
    WithEnvironment("production").
    WithLevel("info").
    WithAppName("my-service").
    WithVersion("2.0.1")

tlog.Init(cfg)
```

Output:
```json
{"timestamp":"2024-12-27T15:04:05.123+07:00","level":"INFO","caller":"main.go:10","message":"Application started","service":"my-service","version":"2.0.1","port":8080}
```

#### With File Output

```go
cfg := tlog.DefaultConfig().
    WithEnvironment("production").
    WithLevel("debug").
    WithAppName("my-api").
    WithVersion("1.2.3").
    WithFile("logs/app.log").
    WithFileRotation(100, 5, 7, true)  // 100MB, 5 backups, 7 days, compress
    
tlog.Init(cfg)
```

#### Both Console and File

```go
cfg := tlog.DefaultConfig().
    WithEnvironment("production").
    WithConsole(true).                  // Keep console output
    WithFile("logs/app.log")            // Also write to file
    
tlog.Init(cfg)
```

#### Custom Timezone

```go
// Use UTC
cfg := tlog.DefaultConfig().
    WithTimezone(time.UTC)

// Use specific timezone
loc, _ := time.LoadLocation("America/New_York")
cfg := tlog.DefaultConfig().
    WithTimezone(loc)
    
tlog.Init(cfg)
```

#### Full Configuration Example

```go
import (
    "time"
    "github.com/thienel/tlog"
)

func main() {
    // Load from environment or config file
    cfg := tlog.Config{
        Environment:   "production",
        Level:         "debug",
        AppName:       "my-service",
        Version:       "2.0.0",
        EnableConsole: true,
        EnableFile:    true,
        FilePath:      "/var/log/my-service/app.log",
        MaxSizeMB:     200,
        MaxBackups:    10,
        MaxAgeDays:    30,
        Compress:      true,
        Timezone:      time.UTC,
    }
    
    if err := tlog.Init(cfg); err != nil {
        panic(err)
    }
    defer tlog.Sync()
}
```

---

## Context-Aware Logging

### Context Keys

tlog provides built-in context keys for request tracing:

- `RequestIDKey` - Request ID for tracing
- `UserIDKey` - Authenticated user ID
- `TraceIDKey` - Distributed trace ID

### Adding Context Values

```go
import (
    "context"
    "github.com/thienel/tlog"
)

func main() {
    ctx := context.Background()
    
    // Add individual values
    ctx = tlog.WithRequestID(ctx, "req-abc-123")
    ctx = tlog.WithUserID(ctx, 42)
    ctx = tlog.WithTraceID(ctx, "trace-xyz-789")
    
    // Or add multiple at once
    ctx = tlog.ContextWithFields(ctx, "req-abc-123", 42, "trace-xyz-789")
}
```

### Logging with Context

```go
// Use context-aware log functions
tlog.InfoCtx(ctx, "Processing request", zap.String("action", "create"))
tlog.ErrorCtx(ctx, "Failed to process", zap.Error(err))
tlog.DebugCtx(ctx, "Debug info")
tlog.WarnCtx(ctx, "Warning message")

// Output includes context fields automatically:
// {"message":"Processing request","request_id":"req-abc-123","user_id":42,"trace_id":"trace-xyz-789","action":"create"}
```

### Getting Logger with Context

```go
// Get logger with context fields pre-attached
logger := tlog.FromContext(ctx)
logger.Info("Custom log", zap.String("extra", "field"))

// Useful in service layers
func (s *UserService) GetUser(ctx context.Context, id uint) (*User, error) {
    log := tlog.FromContext(ctx)
    log.Info("Fetching user", zap.Uint("id", id))
    // ...
}
```

---

## Gin Integration

### Basic Middleware Usage

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/thienel/tlog"
)

func main() {
    tlog.InitWithDefaults()
    defer tlog.Sync()
    
    r := gin.New()
    r.Use(gin.Recovery())
    
    // Add tlog middleware with default options
    r.Use(tlog.GinMiddleware())
    
    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello"})
    })
    
    r.Run(":8080")
}
```

### GinConfig Options

```go
type GinConfig struct {
    RequestIDHeader string   // Header name for request ID (default: "X-Request-ID")
    MaxBodyLogSize  int      // Max body size to log (default: 4096 bytes)
    LogRequestBody  bool     // Log request body on errors (default: true)
    LogResponseBody bool     // Log response body on errors (default: true)
    SkipPaths       []string // Paths to skip logging
    UseUUIDv7       bool     // Use UUID v7 for request IDs (default: true)
}
```

### Gin Middleware Options

```go
// Skip health check and metrics paths
r.Use(tlog.GinMiddleware(
    tlog.WithSkipPaths("/health", "/metrics", "/ready"),
))

// Custom request ID header
r.Use(tlog.GinMiddleware(
    tlog.WithRequestIDHeader("X-Correlation-ID"),
))

// Increase max body log size
r.Use(tlog.GinMiddleware(
    tlog.WithMaxBodyLogSize(8192), // 8KB
))

// Disable request body logging
r.Use(tlog.GinMiddleware(
    tlog.WithLogRequestBody(false),
))

// Disable response body logging
r.Use(tlog.GinMiddleware(
    tlog.WithLogResponseBody(false),
))

// Use UUID v4 instead of v7
r.Use(tlog.GinMiddleware(
    tlog.WithUUIDv7(false),
))

// Full configuration
r.Use(tlog.GinMiddleware(
    tlog.WithRequestIDHeader("X-Request-ID"),
    tlog.WithMaxBodyLogSize(4096),
    tlog.WithLogRequestBody(true),
    tlog.WithLogResponseBody(true),
    tlog.WithSkipPaths("/health", "/metrics"),
    tlog.WithUUIDv7(true),
))
```

### What the Middleware Logs

**Request Received:**
```json
{
    "message": "Request received",
    "request_id": "019405a0-1234-7abc-8def-0123456789ab",
    "method": "POST",
    "path": "/api/users",
    "query": "page=1",
    "client_ip": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
}
```

**Request Completed (Success):**
```json
{
    "message": "Request completed",
    "request_id": "019405a0-1234-7abc-8def-0123456789ab",
    "method": "POST",
    "path": "/api/users",
    "status": 200,
    "latency": "15.234ms",
    "client_ip": "192.168.1.100",
    "user_id": 42,
    "response_size": 256
}
```

**Request Completed (Error >= 400):**
```json
{
    "level": "WARN",
    "message": "Request completed with client error",
    "request_id": "...",
    "status": 400,
    "request_body": "{\"name\":\"\"}",
    "response_body": "{\"error\":\"name is required\"}"
}
```

### Using Request ID in Handlers

```go
func CreateUser(c *gin.Context) {
    // Request ID is automatically set by middleware
    requestID, _ := c.Get("request_id")
    
    // Create context with request ID for service calls
    ctx := tlog.WithRequestID(c.Request.Context(), requestID.(string))
    
    // Add user ID after authentication
    ctx = tlog.WithUserID(ctx, currentUser.ID)
    
    // Log with context
    tlog.InfoCtx(ctx, "Creating user")
    
    // Pass context to services
    user, err := userService.Create(ctx, req)
}
```

---

## GORM Integration

### Basic Usage

```go
import (
    "time"
    "github.com/thienel/tlog"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    tlog.InitWithDefaults()
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: tlog.NewGormLogger(),
    })
    if err != nil {
        tlog.Fatal("Failed to connect database", zap.Error(err))
    }
}
```

### GormConfig Options

```go
type GormConfig struct {
    SlowThreshold        time.Duration   // Threshold for slow query warning (default: 200ms)
    IgnoreRecordNotFound bool            // Skip logging ErrRecordNotFound (default: true)
    LogLevel             logger.LogLevel // GORM log level (default: Warn)
}
```

### GORM Logger Options

```go
import (
    gormlogger "gorm.io/gorm/logger"
)

// Custom slow query threshold
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: tlog.NewGormLogger(
        tlog.WithSlowThreshold(500 * time.Millisecond), // 500ms
    ),
})

// Don't ignore record not found errors
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: tlog.NewGormLogger(
        tlog.WithIgnoreRecordNotFound(false),
    ),
})

// Set GORM log level
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: tlog.NewGormLogger(
        tlog.WithGormLogLevel(gormlogger.Info), // Log all queries
    ),
})

// Full configuration
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: tlog.NewGormLogger(
        tlog.WithSlowThreshold(200 * time.Millisecond),
        tlog.WithIgnoreRecordNotFound(true),
        tlog.WithGormLogLevel(gormlogger.Warn),
    ),
})
```

### GORM Log Level Options

| Level | Description |
|-------|-------------|
| `gormlogger.Silent` | No logging |
| `gormlogger.Error` | Log errors only |
| `gormlogger.Warn` | Log errors and slow queries (default) |
| `gormlogger.Info` | Log all queries |

### What GORM Logger Logs

**Normal Query (Info level):**
```json
{
    "message": "SQL",
    "sql": "SELECT * FROM users WHERE id = 1",
    "rows": 1,
    "elapsed": "1.234ms",
    "caller": "user_repository.go:42",
    "request_id": "req-abc-123"
}
```

**Slow Query (Warn level):**
```json
{
    "level": "WARN",
    "message": "Slow SQL",
    "sql": "SELECT * FROM orders WHERE created_at > ?",
    "rows": 10000,
    "elapsed": "523.456ms",
    "threshold": "200ms",
    "caller": "order_repository.go:78"
}
```

**Error (Error level):**
```json
{
    "level": "ERROR",
    "message": "SQL Error",
    "sql": "INSERT INTO users (email) VALUES (?)",
    "error": "duplicate key value violates unique constraint",
    "caller": "user_repository.go:25"
}
```

### Context-Aware GORM Queries

```go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
    var user User
    
    // Pass context to GORM - request_id will be included in logs
    err := r.db.WithContext(ctx).First(&user, id).Error
    
    return &user, err
}
```

---

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/thienel/tlog"
    "go.uber.org/zap"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    gormlogger "gorm.io/gorm/logger"
)

func main() {
    // Initialize logger
    cfg := tlog.DefaultConfig().
        WithEnvironment("production").
        WithLevel("debug").
        WithAppName("my-api").
        WithVersion("1.0.0").
        WithFile("logs/app.log").
        WithFileRotation(100, 5, 30, true)

    if err := tlog.Init(cfg); err != nil {
        panic(err)
    }
    defer tlog.Sync()

    // Initialize database with GORM logger
    db, err := gorm.Open(postgres.Open("..."), &gorm.Config{
        Logger: tlog.NewGormLogger(
            tlog.WithSlowThreshold(200 * time.Millisecond),
            tlog.WithGormLogLevel(gormlogger.Warn),
        ),
    })
    if err != nil {
        tlog.Fatal("Failed to connect database", zap.Error(err))
    }

    // Initialize Gin
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(tlog.GinMiddleware(
        tlog.WithSkipPaths("/health"),
    ))

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // API endpoint with context logging
    r.GET("/users/:id", func(c *gin.Context) {
        // Get request context with request_id
        requestID, _ := c.Get("request_id")
        ctx := tlog.WithRequestID(c.Request.Context(), requestID.(string))

        // Log with context
        tlog.InfoCtx(ctx, "Fetching user", zap.String("user_id", c.Param("id")))

        // Database query with context
        var user User
        if err := db.WithContext(ctx).First(&user, c.Param("id")).Error; err != nil {
            tlog.ErrorCtx(ctx, "User not found", zap.Error(err))
            c.JSON(404, gin.H{"error": "user not found"})
            return
        }

        c.JSON(200, user)
    })

    tlog.Info("Server starting", zap.String("addr", ":8080"))
    r.Run(":8080")
}
```

---

## Best Practices

1. **Always call `tlog.Sync()` on shutdown** - Ensures all buffered logs are flushed

2. **Use context-aware logging in services** - Pass context from handlers to maintain request tracing

3. **Set appropriate log levels per environment**:
   - Development: `debug`
   - Staging: `debug` or `info`
   - Production: `info` or `warn`

4. **Use structured fields instead of string formatting**:
   ```go
   // Good
   tlog.Info("User created", zap.Uint("user_id", user.ID), zap.String("email", user.Email))
   
   // Bad
   tlog.S().Infof("User created: %d - %s", user.ID, user.Email)
   ```

5. **Configure file rotation in production** - Prevent disk space issues

6. **Skip noisy paths** - Exclude health checks and metrics from logging

---

## License

MIT License

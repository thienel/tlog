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

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/thienel/tlog"
    "go.uber.org/zap"
)

func main() {
    // Initialize with defaults
    if err := tlog.InitWithDefaults(); err != nil {
        panic(err)
    }
    defer tlog.Sync()

    // Log messages
    tlog.Info("Application started")
    tlog.Debug("Debug message", zap.String("key", "value"))
    tlog.Error("Error occurred", zap.Error(err))
}
```

### Custom Configuration

```go
cfg := tlog.DefaultConfig().
    WithEnvironment("production").
    WithLevel("debug").
    WithAppName("my-app").
    WithFile("logs/app.log").
    WithFileRotation(100, 3, 30, true) // 100MB, 3 backups, 30 days, compress

if err := tlog.Init(cfg); err != nil {
    panic(err)
}
```

## Gin Integration

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/thienel/tlog"
)

func main() {
    tlog.InitWithDefaults()
    
    r := gin.New()
    
    // Use tlog middleware
    r.Use(tlog.GinMiddleware(
        tlog.WithSkipPaths("/health", "/metrics"),
        tlog.WithMaxBodyLogSize(8192),
    ))
    
    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello"})
    })
    
    r.Run(":8080")
}
```

### Middleware Features

- Auto-generates `X-Request-ID` header (UUID v7)
- Logs request received and completed events
- Captures request/response body on errors (>= 400)
- Status-based log levels (INFO/WARN/ERROR)
- Skip configurable paths

## GORM Integration

```go
import (
    "github.com/thienel/tlog"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    tlog.InitWithDefaults()
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: tlog.NewGormLogger(
            tlog.WithSlowThreshold(200 * time.Millisecond),
            tlog.WithIgnoreRecordNotFound(true),
        ),
    })
    if err != nil {
        tlog.Fatal("Failed to connect database", zap.Error(err))
    }
}
```

### GORM Logger Features

- Slow query detection with configurable threshold
- Ignores `ErrRecordNotFound` by default
- Context-aware (includes `request_id`, `trace_id`)

## Context-Aware Logging

```go
import (
    "context"
    "github.com/thienel/tlog"
    "go.uber.org/zap"
)

func handler(c *gin.Context) {
    ctx := c.Request.Context()
    
    // Add context values
    ctx = tlog.WithRequestID(ctx, "req-123")
    ctx = tlog.WithUserID(ctx, 42)
    
    // Log with context
    tlog.InfoCtx(ctx, "Processing request")
    
    // Or get logger with context fields
    logger := tlog.FromContext(ctx)
    logger.Info("Custom log", zap.String("extra", "field"))
}
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `Environment` | `"development"` | Log format (`development`/`production`) |
| `Level` | `"info"` | Minimum log level |
| `AppName` | `"app"` | Logger name in structured logs |
| `EnableConsole` | `true` | Enable stdout output |
| `EnableFile` | `false` | Enable file output |
| `FilePath` | `"logs/app.log"` | Log file path |
| `MaxSizeMB` | `100` | Max file size before rotation |
| `MaxBackups` | `3` | Number of old files to keep |
| `MaxAgeDays` | `30` | Max days to keep files |
| `Compress` | `true` | Compress rotated files |
| `Timezone` | `Asia/Ho_Chi_Minh` | Timezone for timestamps |

## License

MIT License

package tlog

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GinConfig contains configuration for the Gin middleware.
type GinConfig struct {
	// RequestIDHeader is the header key for request ID.
	// Default: "X-Request-ID"
	RequestIDHeader string

	// MaxBodyLogSize limits the size of request/response body to log.
	// Default: 4096 bytes
	MaxBodyLogSize int

	// LogRequestBody enables logging request body on errors (>= 400).
	// Default: true
	LogRequestBody bool

	// LogResponseBody enables logging response body on errors (>= 400).
	// Default: true
	LogResponseBody bool

	// SkipPaths is a list of paths to skip logging.
	SkipPaths []string

	// UseUUIDv7 uses UUID v7 (time-ordered) for request IDs.
	// Default: true
	UseUUIDv7 bool
}

// DefaultGinConfig returns a GinConfig with sensible defaults.
func DefaultGinConfig() GinConfig {
	return GinConfig{
		RequestIDHeader: "X-Request-ID",
		MaxBodyLogSize:  4096,
		LogRequestBody:  true,
		LogResponseBody: true,
		SkipPaths:       nil,
		UseUUIDv7:       true,
	}
}

// GinOptionFunc is a function that configures GinConfig.
type GinOptionFunc func(*GinConfig)

// WithRequestIDHeader sets the request ID header name.
func WithRequestIDHeader(header string) GinOptionFunc {
	return func(c *GinConfig) {
		c.RequestIDHeader = header
	}
}

// WithMaxBodyLogSize sets the maximum body size to log.
func WithMaxBodyLogSize(size int) GinOptionFunc {
	return func(c *GinConfig) {
		c.MaxBodyLogSize = size
	}
}

// WithLogRequestBody enables/disables request body logging.
func WithLogRequestBody(enabled bool) GinOptionFunc {
	return func(c *GinConfig) {
		c.LogRequestBody = enabled
	}
}

// WithLogResponseBody enables/disables response body logging.
func WithLogResponseBody(enabled bool) GinOptionFunc {
	return func(c *GinConfig) {
		c.LogResponseBody = enabled
	}
}

// WithSkipPaths sets paths to skip logging.
func WithSkipPaths(paths ...string) GinOptionFunc {
	return func(c *GinConfig) {
		c.SkipPaths = paths
	}
}

// WithUUIDv7 enables/disables UUID v7 for request IDs.
func WithUUIDv7(enabled bool) GinOptionFunc {
	return func(c *GinConfig) {
		c.UseUUIDv7 = enabled
	}
}

// responseWriter wraps gin.ResponseWriter to capture response body.
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// GinMiddleware returns a Gin middleware that logs HTTP requests.
func GinMiddleware(opts ...GinOptionFunc) gin.HandlerFunc {
	cfg := DefaultGinConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	skipPathMap := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPathMap[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip logging for configured paths
		if skipPathMap[path] {
			c.Next()
			return
		}

		// Generate or get request ID
		requestID := c.GetHeader(cfg.RequestIDHeader)
		if requestID == "" {
			if cfg.UseUUIDv7 {
				requestID = uuid.Must(uuid.NewV7()).String()
			} else {
				requestID = uuid.New().String()
			}
		}
		c.Set("request_id", requestID)
		c.Header(cfg.RequestIDHeader, requestID)

		// Request start time
		start := time.Now()

		// Get request info
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Read request body for non-GET requests (for error debugging)
		var requestBody string
		if cfg.LogRequestBody && method != "GET" && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// Restore the body for handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				if len(bodyBytes) > 0 {
					if len(bodyBytes) > cfg.MaxBodyLogSize {
						requestBody = string(bodyBytes[:cfg.MaxBodyLogSize]) + "...[truncated]"
					} else {
						requestBody = string(bodyBytes)
					}
				}
			}
		}

		// Log request received
		L().Info("Request received",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		)

		// Wrap response writer to capture response body
		blw := &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Get user ID if authenticated
		var userID uint
		if id, exists := c.Get("user_id"); exists {
			if uid, ok := id.(uint); ok {
				userID = uid
			}
		}

		// Build log fields
		logFields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		}

		if query != "" {
			logFields = append(logFields, zap.String("query", query))
		}

		if userID > 0 {
			logFields = append(logFields, zap.Uint("user_id", userID))
		}

		// Add response size
		responseSize := c.Writer.Size()
		if responseSize > 0 {
			logFields = append(logFields, zap.Int("response_size", responseSize))
		}

		// For error responses, include response body and request body
		if statusCode >= 400 {
			if cfg.LogResponseBody {
				responseBody := blw.body.String()
				if len(responseBody) > cfg.MaxBodyLogSize {
					responseBody = responseBody[:cfg.MaxBodyLogSize] + "...[truncated]"
				}
				logFields = append(logFields, zap.String("response_body", responseBody))
			}

			if cfg.LogRequestBody && requestBody != "" {
				logFields = append(logFields, zap.String("request_body", requestBody))
			}
		}

		// Add gin errors if exists
		if len(c.Errors) > 0 {
			logFields = append(logFields, zap.String("gin_errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			L().Error("Request completed with server error", logFields...)
		case statusCode >= 400:
			L().Warn("Request completed with client error", logFields...)
		default:
			L().Info("Request completed", logFields...)
		}
	}
}

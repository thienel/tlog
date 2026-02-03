package tlog

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DefaultMaskValue is the default replacement for masked fields.
const DefaultMaskValue = "******"

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

	// MaskPatterns is a list of compiled regex patterns for field names to mask.
	// Values of fields whose names match any pattern will be replaced with "******".
	MaskPatterns []*regexp.Regexp
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

// WithMaskPatterns sets regex patterns for field names to mask in request/response bodies.
// Values of JSON fields whose names match any pattern will be replaced with "******".
// Example: WithMaskPatterns(`(?i)password`, `(?i)secret`, `(?i)token`)
func WithMaskPatterns(patterns ...string) GinOptionFunc {
	return func(c *GinConfig) {
		c.MaskPatterns = make([]*regexp.Regexp, 0, len(patterns))
		for _, p := range patterns {
			if re, err := regexp.Compile(p); err == nil {
				c.MaskPatterns = append(c.MaskPatterns, re)
			}
		}
	}
}

// maskJSONFields masks values of fields whose names match any of the mask patterns.
// It recursively processes nested objects and arrays.
func maskJSONFields(data any, patterns []*regexp.Regexp) any {
	if len(patterns) == 0 {
		return data
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			if shouldMask(key, patterns) {
				result[key] = DefaultMaskValue
			} else {
				result[key] = maskJSONFields(val, patterns)
			}
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = maskJSONFields(item, patterns)
		}
		return result
	default:
		return data
	}
}

// shouldMask checks if a field name matches any of the mask patterns.
func shouldMask(fieldName string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(fieldName) {
			return true
		}
	}
	return false
}

// maskBody parses JSON body and masks sensitive fields, returning the masked JSON string.
// If parsing fails, returns the original body unchanged.
func maskBody(body string, patterns []*regexp.Regexp) string {
	if len(patterns) == 0 || body == "" {
		return body
	}

	var data any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return body
	}

	masked := maskJSONFields(data, patterns)
	result, err := json.Marshal(masked)
	if err != nil {
		return body
	}

	return string(result)
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
		protocol := c.Request.Proto
		host := c.Request.Host

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
			zap.Int("status_code", statusCode),
			zap.Int64("duration_ms", latency.Milliseconds()),
			zap.String("ip_address", clientIP),
			zap.String("protocol", protocol),
			zap.String("host", host),
		}

		if query != "" {
			logFields = append(logFields, zap.String("query_string", query))
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
				responseBody = maskBody(responseBody, cfg.MaskPatterns)
				logFields = append(logFields, zap.String("response_body", responseBody))
			}

			if cfg.LogRequestBody && requestBody != "" {
				maskedRequestBody := maskBody(requestBody, cfg.MaskPatterns)
				logFields = append(logFields, zap.String("request_body", maskedRequestBody))
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

package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

type Options struct {
	Logger      *zap.Logger
	ServiceName string
	StartedAt   time.Time
	Version     version.Info
	Checkers    []HealthChecker
}

func NewEngine(opts Options) *gin.Engine {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if opts.StartedAt.IsZero() {
		opts.StartedAt = time.Now()
	}
	if opts.ServiceName == "" {
		opts.ServiceName = "licenseiq"
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestIDMiddleware())
	engine.Use(zapRequestLogger(opts.Logger))
	engine.Use(otelgin.Middleware(opts.ServiceName))

	health := NewHealthHandler(opts.StartedAt, opts.Version, opts.Checkers...)

	engine.GET("/health", health.Liveness)
	engine.GET("/ready", health.Readiness)
	engine.GET("/metrics", Metrics())

	return engine
}

func NewServer(addr string, handler http.Handler, readTimeout, writeTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if strings.TrimSpace(requestID) == "" {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

func zapRequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		requestID, _ := c.Get("request_id")
		fields := []zap.Field{
			zap.String("request_id", fmt.Sprint(requestID)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			logger.Error("request", append(fields, zap.String("error", c.Errors.String()))...)
			return
		}
		logger.Info("request", fields...)
	}
}

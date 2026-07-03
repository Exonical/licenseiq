package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

type HealthHandler struct {
	startedAt time.Time
	version   version.Info
	checkers  []HealthChecker
}

type healthCheckResult struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type healthResponse struct {
	Status string              `json:"status"`
	Checks []healthCheckResult `json:"checks,omitempty"`
}

type livenessResponse struct {
	Status    string       `json:"status"`
	Version   version.Info `json:"version"`
	UptimeSec float64      `json:"uptime_seconds"`
}

func NewHealthHandler(startedAt time.Time, versionInfo version.Info, checkers ...HealthChecker) *HealthHandler {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	return &HealthHandler{
		startedAt: startedAt,
		version:   versionInfo,
		checkers:  checkers,
	}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, livenessResponse{
		Status:    "ok",
		Version:   h.version,
		UptimeSec: time.Since(h.startedAt).Seconds(),
	})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	checks := make([]healthCheckResult, 0, len(h.checkers))
	ok := true
	for _, checker := range h.checkers {
		result := healthCheckResult{Name: checker.Name()}
		if err := checker.Check(c.Request.Context()); err != nil {
			result.Error = err.Error()
			ok = false
		} else {
			result.OK = true
		}
		checks = append(checks, result)
	}
	status := "ok"
	code := http.StatusOK
	if !ok {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, healthResponse{
		Status: status,
		Checks: checks,
	})
}

func Metrics() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

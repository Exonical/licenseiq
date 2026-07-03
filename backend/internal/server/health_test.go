package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/version"
	"github.com/gin-gonic/gin"
)

type fakeChecker struct {
	name string
	err  error
}

func (f fakeChecker) Name() string                { return f.name }
func (f fakeChecker) Check(context.Context) error { return f.err }

func TestHealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := NewEngine(Options{
		ServiceName: "licenseiq",
		StartedAt:   time.Now().Add(-5 * time.Minute),
		Version:     version.Current(),
		Checkers: []HealthChecker{
			fakeChecker{name: "db"},
			fakeChecker{name: "cache"},
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("health status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("ready status = %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestReadyReturns503OnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := NewEngine(Options{
		ServiceName: "licenseiq",
		StartedAt:   time.Now(),
		Version:     version.Current(),
		Checkers: []HealthChecker{
			fakeChecker{name: "db"},
			fakeChecker{name: "cache", err: context.Canceled},
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, body=%s", rr.Code, rr.Body.String())
	}
}

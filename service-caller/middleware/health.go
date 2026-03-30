package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	isShuttingDown atomic.Bool
)

func SetShuttingDown(val bool) {
	isShuttingDown.Store(val)
}

func IsShuttingDown() bool {
	return isShuttingDown.Load()
}

type DependencyChecker interface {
	CheckHealth() (bool, string)
}

type HealthHandler struct {
	serviceName  string
	version      string
	dependencies []DependencyChecker
}

func NewHealthHandler(serviceName, version string, deps ...DependencyChecker) *HealthHandler {
	return &HealthHandler{
		serviceName:  serviceName,
		version:      version,
		dependencies: deps,
	}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   h.serviceName,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	if IsShuttingDown() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not_ready",
			"service":   h.serviceName,
			"reason":    "shutting_down",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	dependencyStatus := make(map[string]string)
	allHealthy := true

	for _, dep := range h.dependencies {
		healthy, _ := dep.CheckHealth()
		if healthy {
			dependencyStatus["dependencies"] = "healthy"
		} else {
			dependencyStatus["dependencies"] = "unhealthy"
			allHealthy = false
		}
		break
	}

	if len(h.dependencies) == 0 {
		dependencyStatus["self"] = "healthy"
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":       status,
		"service":      h.serviceName,
		"version":      h.version,
		"dependencies": dependencyStatus,
		"timestamp":    time.Now().Format(time.RFC3339),
	})
}

type VersionHandler struct {
	version   string
	gitCommit string
	buildTime string
}

func NewVersionHandler(version, gitCommit, buildTime string) *VersionHandler {
	return &VersionHandler{
		version:   version,
		gitCommit: gitCommit,
		buildTime: buildTime,
	}
}

func (v *VersionHandler) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    v.version,
		"git_commit": v.gitCommit,
		"build_time": v.buildTime,
	})
}

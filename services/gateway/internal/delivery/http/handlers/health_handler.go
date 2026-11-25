package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const readinessTimeout = 2 * time.Second

type HealthHandler struct {
	deps map[string]*grpc.ClientConn
}

func NewHealthHandler(deps map[string]*grpc.ClientConn) *HealthHandler {
	return &HealthHandler{deps: deps}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()

	checks := make(map[string]string, len(h.deps))
	ready := true
	for name, conn := range h.deps {
		state := waitReady(ctx, conn)
		checks[name] = state.String()
		if state != connectivity.Ready {
			ready = false
		}
	}

	if !ready {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "checks": checks})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "checks": checks})
}

func waitReady(ctx context.Context, conn *grpc.ClientConn) connectivity.State {
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return state
		}
		if state == connectivity.Idle {
			conn.Connect()
		}
		if !conn.WaitForStateChange(ctx, state) {
			return conn.GetState()
		}
	}
}

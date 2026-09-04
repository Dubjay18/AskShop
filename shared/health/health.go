// Package health provides a consistent /healthz endpoint across services,
// for use by container orchestrators (Kubernetes liveness/readiness probes,
// docker-compose healthchecks) and manual checks alike.
package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register adds GET /healthz to router, reporting the given service name.
func Register(router *gin.Engine, serviceName string) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": serviceName, "status": "ok"})
	})
}

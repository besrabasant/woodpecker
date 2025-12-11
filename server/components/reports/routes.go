package reports

import (
	"github.com/gin-gonic/gin"

	"go.woodpecker-ci.org/woodpecker/v3/server/router/middleware/session"
)

// Routes registers report related routes on the provided router group.
func Routes(repo *gin.RouterGroup) {
	repo.GET("/pipelines/:number/reports", session.MustPull, GetPipelineReports)
}

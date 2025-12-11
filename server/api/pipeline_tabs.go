package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go.woodpecker-ci.org/woodpecker/v3/server"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/router/middleware/session"
)

// GetPipelineTabs returns the configured dynamic tabs for a pipeline.
//
//	@Summary	Get dynamic pipeline tabs
//	@Router		/repos/{repo_id}/pipelines/{number}/tabs [get]
//	@Produce	json
//	@Param		Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param		repo_id			path	int		true	"repository id"
//	@Param		number			path	int		true	"pipeline number"
//	@Tags		Pipelines
func GetPipelineTabs(c *gin.Context) {
	perm := session.Perm(c)
	tabs := filterPipelineTabs(server.Config.PipelineTabs, perm)
	c.JSON(http.StatusOK, tabs)
}

func filterPipelineTabs(tabs []model.PipelineTabDefinition, perm *model.Perm) []model.PipelineTabDefinition {
	if perm == nil {
		return nil
	}
	result := make([]model.PipelineTabDefinition, 0, len(tabs))
	for _, tab := range tabs {
		if tab.RequiresPush && !perm.Push {
			continue
		}
		result = append(result, tab)
	}
	return result
}

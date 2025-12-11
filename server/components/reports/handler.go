package reports

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go.woodpecker-ci.org/woodpecker/v3/server/router/middleware/session"
)

// GetPipelineReports returns an HTML snippet with pipeline reports information.
func GetPipelineReports(c *gin.Context) {
	repo := session.Repo(c)
	number, err := strconv.ParseInt(c.Param("number"), 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	html := fmt.Sprintf(
		`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<title>Reports for %s #%d</title>
<style>
body { font-family: sans-serif; margin: 1rem; color: #1f2533; }
.subtitle { color: #6b7280; font-size: 0.9rem; }
</style>
</head>
<body>
  <h3>Reports for %s #%d</h3>
  <p class="subtitle">Generated at %s</p>
  <p>No custom reports are available yet. This placeholder will be replaced with server-provided reports.</p>
</body>
</html>`,
		repo.FullName,
		number,
		repo.FullName,
		number,
		time.Now().Format(time.RFC1123),
	)

	c.Header("X-Frame-Options", "SAMEORIGIN")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

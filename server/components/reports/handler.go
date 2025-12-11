package reports

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"

	"go.woodpecker-ci.org/woodpecker/v3/server"
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

	html, contentType := buildFallbackHTML(repo.FullName, number)

	switch baseURL := strings.TrimSpace(server.Config.Components.Reports.BaseURL); {
	case server.Config.Components.Reports.Endpoint != "" &&
		server.Config.Components.Reports.AccessKey != "" &&
		server.Config.Components.Reports.SecretKey != "" &&
		server.Config.Components.Reports.Bucket != "":
		if reportHTML, ct, err := fetchReportFromS3(
			server.Config.Components.Reports.Endpoint,
			server.Config.Components.Reports.AccessKey,
			server.Config.Components.Reports.SecretKey,
			server.Config.Components.Reports.Bucket,
			string(repo.ForgeRemoteID),
			number,
		); err == nil {
			html = reportHTML
			if ct != "" {
				contentType = ct
			}
		} else {
			log.Error().Err(err).Msg("failed to load reports from s3 source, falling back")
		}
	case baseURL != "":
		if reportHTML, ct, err := fetchReportHTML(baseURL, string(repo.ForgeRemoteID), number); err == nil {
			html = reportHTML
			if ct != "" {
				contentType = ct
			}
		} else {
			log.Error().Err(err).Str("base_url", baseURL).Msg("failed to load reports from HTTP source, falling back")
		}
	default:
		log.Debug().Msg("no report source configured, serving fallback html")
	}

	c.Header("X-Frame-Options", "SAMEORIGIN")
	c.Data(http.StatusOK, contentType, []byte(html))
}

func buildFallbackHTML(repoName string, pipelineNumber int64) (string, string) {
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
  <h3>Reports for %s #%d are have not been generated yet or do not exist.</h3>
</body>
</html>`,
		repoName,
		pipelineNumber,
		repoName,
		pipelineNumber,
	)

	return html, "text/html; charset=utf-8"
}

func fetchReportHTML(baseURL string, forgeRemoteID string, pipelineNumber int64) (string, string, error) {
	url := fmt.Sprintf("%s/repos/%s/builds/%d/index.html", strings.TrimSuffix(baseURL, "/"), forgeRemoteID, pipelineNumber)

	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return string(body), resp.Header.Get("Content-Type"), nil
}

func fetchReportFromS3(endpoint, accessKey, secretKey, bucket, forgeRemoteID string, pipelineNumber int64) (string, string, error) {
	client, err := minio.New(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: strings.HasPrefix(endpoint, "https"),
	})
	if err != nil {
		return "", "", err
	}

	objectPath := fmt.Sprintf("repos/%s/builds/%d/artifacts/playwright-report/index.html", forgeRemoteID, pipelineNumber)
	reader, err := client.GetObject(context.Background(), bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return "", "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}

	return string(data), "text/html; charset=utf-8", nil
}

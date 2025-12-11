package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go.woodpecker-ci.org/woodpecker/v3/server"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
)

const defaultManualActionTimeout = 30 * time.Second

type manualActionTriggerRequest struct {
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    map[string]string `json:"body"`
}

type manualActionHTTPRequest struct {
	Method  string
	URL     string
	Body    []byte
	Headers http.Header
	Timeout time.Duration
}

// GetManualActions returns the list of configured manual action definitions.
func GetManualActions(c *gin.Context) {
	if server.Config.ManualActions == nil {
		c.JSON(http.StatusOK, []model.ManualActionDefinition{})
		return
	}
	c.JSON(http.StatusOK, server.Config.ManualActions)
}

// TriggerManualAction executes a configured manual action.
//
//	@Summary	Trigger a manual webhook action
//	@Router		/repos/{repo_id}/manual-actions/{actionId} [post]
//	@Produce	json
//	@Param		Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param		repo_id			path	int		true	"the repository id"
//	@Param		actionId		path	string	true	"manual action identifier"
//	@Param		payload			body	manualActionTriggerRequest	false	"user supplied values for the manual action"
//	@Tags		Pipelines
func TriggerManualAction(c *gin.Context) {
	definition, ok := findManualActionDefinition(c.Param("actionId"))
	if !ok {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	trigger, err := decodeManualActionPayload(c)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	triggerManualActionResponse(c, definition, trigger)
}

// TriggerWebhookAction executes a configured manual action by id using only default values.
//
//	@Summary	Trigger a configured webhook action
//	@Router		/repos/{repo_id}/webhooks/{actionId}/trigger [post]
//	@Produce	json
//	@Param		Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param		repo_id			path	int		true	"the repository id"
//	@Param		actionId		path	string	true	"webhook action identifier"
//	@Tags		Pipelines
func TriggerWebhookAction(c *gin.Context) {
	definition, ok := findManualActionDefinition(c.Param("actionId"))
	if !ok {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	triggerManualActionResponse(c, definition, &manualActionTriggerRequest{})
}

func findManualActionDefinition(id string) (*model.ManualActionDefinition, bool) {
	for i := range server.Config.ManualActions {
		if server.Config.ManualActions[i].ID == id {
			return &server.Config.ManualActions[i], true
		}
	}
	return nil, false
}

func buildManualActionHTTPRequest(definition *model.ManualActionDefinition, payload *manualActionTriggerRequest) (*manualActionHTTPRequest, error) {
	reqURL, err := url.Parse(definition.Request.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid manual action url: %w", err)
	}

	applySectionValues := func(fields []model.ManualActionField, values map[string]string, setter func(key, value string)) error {
		for _, field := range fields {
			value := ""
			if values != nil {
				value = strings.TrimSpace(values[field.Key])
			}
			if value == "" {
				value = strings.TrimSpace(field.Default)
			}
			if value == "" {
				if field.Required {
					return fmt.Errorf("missing required value for %s", field.Key)
				}
				continue
			}
			setter(field.Key, value)
		}
		return nil
	}

	query := reqURL.Query()
	if err := applySectionValues(definition.Request.Query, payload.Query, func(key, value string) {
		query.Set(key, value)
	}); err != nil {
		return nil, err
	}
	reqURL.RawQuery = query.Encode()

	headers := make(http.Header)
	if err := applySectionValues(definition.Request.Headers, payload.Headers, func(key, value string) {
		headers.Set(key, value)
	}); err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if len(definition.Request.Body) > 0 && definition.Request.Method != http.MethodGet {
		bodyPayload := make(map[string]string)
		if err := applySectionValues(definition.Request.Body, payload.Body, func(key, value string) {
			bodyPayload[key] = value
		}); err != nil {
			return nil, err
		}
		if len(bodyPayload) > 0 {
			bodyBytes, err = json.Marshal(bodyPayload)
			if err != nil {
				return nil, fmt.Errorf("marshal manual action body: %w", err)
			}
			if headers.Get("Content-Type") == "" {
				headers.Set("Content-Type", "application/json")
			}
		}
	}

	timeout := defaultManualActionTimeout
	if definition.Request.Timeout != "" {
		if parsed, err := time.ParseDuration(definition.Request.Timeout); err == nil {
			timeout = parsed
		}
	}

	return &manualActionHTTPRequest{
		Method:  definition.Request.Method,
		URL:     reqURL.String(),
		Body:    bodyBytes,
		Headers: headers,
		Timeout: timeout,
	}, nil
}

func executeManualActionRequest(ctx context.Context, req *manualActionHTTPRequest) (int, string, error) {
	var bodyReader io.Reader
	if len(req.Body) > 0 {
		bodyReader = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return 0, "", fmt.Errorf("create manual action request: %w", err)
	}
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: req.Timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return 0, "", fmt.Errorf("execute manual action request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(respBody), nil
}

func triggerManualActionResponse(c *gin.Context, definition *model.ManualActionDefinition, trigger *manualActionTriggerRequest) {
	httpReq, err := buildManualActionHTTPRequest(definition, trigger)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	status, respBody, err := executeManualActionRequest(c.Request.Context(), httpReq)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadGateway, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"success": status >= http.StatusOK && status < 300,
		"body":    respBody,
	})
}

func decodeManualActionPayload(c *gin.Context) (*manualActionTriggerRequest, error) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		return &manualActionTriggerRequest{}, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, err
	}

	trigger := &manualActionTriggerRequest{}
	for key, value := range raw {
		switch key {
		case "query":
			if err := json.Unmarshal(value, &trigger.Query); err != nil {
				return nil, err
			}
		case "headers":
			if err := json.Unmarshal(value, &trigger.Headers); err != nil {
				return nil, err
			}
		case "body":
			if err := json.Unmarshal(value, &trigger.Body); err != nil {
				return nil, err
			}
		default:
			if trigger.Body == nil {
				trigger.Body = make(map[string]string)
			}
			var strValue string
			if err := json.Unmarshal(value, &strValue); err == nil {
				trigger.Body[key] = strValue
				continue
			}

			var generic interface{}
			if err := json.Unmarshal(value, &generic); err != nil {
				return nil, err
			}
			trigger.Body[key] = fmt.Sprint(generic)
		}
	}

	return trigger, nil
}

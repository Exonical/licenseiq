package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
	"go.uber.org/zap"
)

type httpClient struct {
	baseURL     *url.URL
	client      *http.Client
	deployment  string
	email       string
	apiToken    string
	personalTok string
	projectKey  string
	issueType   string
	logger      *zap.Logger
}

func NewClient(cfg config.JiraConfig, logger *zap.Logger) (Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	baseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse jira base url: %w", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("jira base url must include scheme and host")
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}
	return &httpClient{
		baseURL:     baseURL,
		client:      &http.Client{Timeout: cfg.HTTPTimeout},
		deployment:  strings.ToLower(strings.TrimSpace(cfg.Deployment)),
		email:       strings.TrimSpace(cfg.Email),
		apiToken:    cfg.APIToken,
		personalTok: cfg.PersonalToken,
		projectKey:  strings.TrimSpace(cfg.ProjectKey),
		issueType:   strings.TrimSpace(cfg.IssueType),
		logger:      logger,
	}, nil
}

func (c *httpClient) CreateIssue(ctx context.Context, req CreateIssueRequest) (*CreateIssueResponse, error) {
	payload := map[string]any{
		"fields": map[string]any{
			"project":     map[string]any{"key": req.ProjectKey},
			"summary":     req.Summary,
			"description": req.Description,
			"issuetype":   map[string]any{"name": req.IssueType},
		},
	}
	var resp createIssueWireResponse
	if err := c.doJSON(ctx, http.MethodPost, "/rest/api/2/issue", payload, &resp); err != nil {
		return nil, err
	}
	out := &CreateIssueResponse{Key: resp.Key, URL: resp.Self, Status: resp.Fields.Status.Name}
	if out.Status == "" {
		out.Status = "Created"
	}
	return out, nil
}

func (c *httpClient) TransitionIssue(ctx context.Context, req TransitionIssueRequest) error {
	if strings.TrimSpace(req.Status) == "" {
		return fmt.Errorf("status is required")
	}
	var transitions transitionListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/rest/api/2/issue/"+url.PathEscape(req.IssueKey)+"/transitions", nil, &transitions); err != nil {
		return err
	}
	transitionID := ""
	for _, tr := range transitions.Transitions {
		if strings.EqualFold(tr.Name, req.Status) || strings.EqualFold(tr.ID, req.Status) {
			transitionID = tr.ID
			break
		}
	}
	if transitionID == "" {
		return fmt.Errorf("transition %q not found", req.Status)
	}
	payload := map[string]any{"transition": map[string]any{"id": transitionID}}
	return c.doJSON(ctx, http.MethodPost, "/rest/api/2/issue/"+url.PathEscape(req.IssueKey)+"/transitions", payload, nil)
}

func (c *httpClient) LinkIssue(ctx context.Context, req LinkIssueRequest) error {
	payload := map[string]any{"body": req.Comment}
	return c.doJSON(ctx, http.MethodPost, "/rest/api/2/issue/"+url.PathEscape(req.IssueKey)+"/comment", payload, nil)
}

func (c *httpClient) AttachFile(ctx context.Context, req AttachFileRequest) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", req.Filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(req.Data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	return c.doRaw(ctx, http.MethodPost, "/rest/api/2/issue/"+url.PathEscape(req.IssueKey)+"/attachments", "multipart/form-data; boundary="+mw.Boundary(), &buf, map[string]string{"X-Atlassian-Token": "no-check"}, nil)
}

type createIssueWireResponse struct {
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields struct {
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}

type transitionListResponse struct {
	Transitions []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"transitions"`
}

func (c *httpClient) doJSON(ctx context.Context, method, apiPath string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	return c.doRaw(ctx, method, apiPath, "application/json", body, nil, out)
}

func (c *httpClient) doRaw(ctx context.Context, method, apiPath, contentType string, body io.Reader, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.resolvePath(apiPath), body)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.applyAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if len(strings.TrimSpace(string(limited))) > 0 {
			return fmt.Errorf("jira %s %s failed: %s: %s", method, apiPath, resp.Status, strings.TrimSpace(string(limited)))
		}
		return fmt.Errorf("jira %s %s failed: %s", method, apiPath, resp.Status)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *httpClient) applyAuth(req *http.Request) {
	switch c.deployment {
	case "datacenter":
		req.Header.Set("Authorization", "Bearer "+c.personalTok)
	default:
		req.SetBasicAuth(c.email, c.apiToken)
	}
}

func (c *httpClient) resolvePath(apiPath string) string {
	base := *c.baseURL
	base.Path = path.Join(strings.TrimRight(base.Path, "/"), apiPath)
	return base.String()
}

func ProjectKey(c Client, fallback string) string {
	if hc, ok := c.(*httpClient); ok && hc.projectKey != "" {
		return hc.projectKey
	}
	return fallback
}

func IssueType(c Client, fallback string) string {
	if hc, ok := c.(*httpClient); ok && hc.issueType != "" {
		return hc.issueType
	}
	return fallback
}

func (c *httpClient) Logger() *zap.Logger { return c.logger }

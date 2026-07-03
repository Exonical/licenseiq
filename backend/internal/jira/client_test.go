package jira

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
)

func TestNewClientDisabledReturnsNil(t *testing.T) {
	client, err := NewClient(config.JiraConfig{}, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client != nil {
		t.Fatalf("expected nil client")
	}
}

func TestClientCloudBasicAuthAndOperations(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		if user, pass, ok := r.BasicAuth(); !ok || user != "user@example.com" || pass != "token-123" {
			t.Fatalf("unexpected basic auth: %v %v %v", user, pass, ok)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue":
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("unexpected content-type: %s", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			fields := payload["fields"].(map[string]any)
			if fields["summary"] != "Renewal" || fields["issuetype"].(map[string]any)["name"] != "Task" {
				t.Fatalf("unexpected issue payload: %#v", payload)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"key":"ABC-1","self":"https://jira.example.com/rest/api/2/issue/ABC-1","fields":{"status":{"name":"Open"}}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/issue/ABC-1/transitions":
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{{"id": "31", "name": "Done"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue/ABC-1/transitions":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode transition: %v", err)
			}
			transition := payload["transition"].(map[string]any)
			if transition["id"] != "31" {
				t.Fatalf("unexpected transition: %#v", payload)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue/ABC-1/comment":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode comment: %v", err)
			}
			if payload["body"] != "linked" {
				t.Fatalf("unexpected comment payload: %#v", payload)
			}
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue/ABC-1/attachments":
			if got := r.Header.Get("X-Atlassian-Token"); got != "no-check" {
				t.Fatalf("unexpected atlassian token header: %s", got)
			}
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
				t.Fatalf("unexpected content-type: %s", r.Header.Get("Content-Type"))
			}
			mr, err := multipart.NewReader(r.Body, strings.TrimPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=")).ReadForm(16 << 20)
			if err != nil {
				t.Fatalf("read multipart: %v", err)
			}
			if got := mr.File["file"]; len(got) != 1 || got[0].Filename != "note.txt" {
				t.Fatalf("unexpected attachment form: %#v", mr.File)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(config.JiraConfig{
		Enabled:     true,
		BaseURL:     server.URL,
		Deployment:  "cloud",
		Email:       "user@example.com",
		APIToken:    "token-123",
		ProjectKey:  "ABC",
		IssueType:   "Task",
		HTTPTimeout: 5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	created, err := client.CreateIssue(context.Background(), CreateIssueRequest{ProjectKey: "ABC", IssueType: "Task", Summary: "Renewal", Description: "body"})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.Key != "ABC-1" || created.Status != "Open" {
		t.Fatalf("unexpected create response: %#v", created)
	}
	if err := client.TransitionIssue(context.Background(), TransitionIssueRequest{IssueKey: "ABC-1", Status: "Done"}); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if err := client.LinkIssue(context.Background(), LinkIssueRequest{IssueKey: "ABC-1", Comment: "linked"}); err != nil {
		t.Fatalf("link issue: %v", err)
	}
	if err := client.AttachFile(context.Background(), AttachFileRequest{IssueKey: "ABC-1", Filename: "note.txt", ContentType: "text/plain", Data: []byte("hello")}); err != nil {
		t.Fatalf("attach file: %v", err)
	}
	if len(seen) != 5 {
		t.Fatalf("unexpected requests: %#v", seen)
	}
}

func TestClientDatacenterBearerAndNon2xx(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `boom`)
	}))
	defer server.Close()

	client, err := NewClient(config.JiraConfig{
		Enabled:       true,
		BaseURL:       server.URL,
		Deployment:    "datacenter",
		PersonalToken: "pat-secret",
		ProjectKey:    "ABC",
		IssueType:     "Task",
		HTTPTimeout:   5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.CreateIssue(context.Background(), CreateIssueRequest{ProjectKey: "ABC", IssueType: "Task", Summary: "Renewal"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), "pat-secret") {
		t.Fatalf("error leaked secret: %v", err)
	}
	if authHeader != "Bearer pat-secret" {
		t.Fatalf("unexpected bearer auth: %s", authHeader)
	}
}

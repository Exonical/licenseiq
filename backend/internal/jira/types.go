package jira

import "context"

type Client interface {
	CreateIssue(context.Context, CreateIssueRequest) (*CreateIssueResponse, error)
	TransitionIssue(context.Context, TransitionIssueRequest) error
	LinkIssue(context.Context, LinkIssueRequest) error
	AttachFile(context.Context, AttachFileRequest) error
}

type CreateIssueRequest struct {
	ProjectKey  string
	IssueType   string
	Summary     string
	Description string
}

type CreateIssueResponse struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

type TransitionIssueRequest struct {
	IssueKey string
	Status   string
}

type LinkIssueRequest struct {
	IssueKey string
	Comment  string
}

type AttachFileRequest struct {
	IssueKey    string
	Filename    string
	ContentType string
	Data        []byte
}

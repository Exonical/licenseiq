package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type slackNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewSlackNotifier(webhookURL string, client *http.Client) Notifier {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" || client == nil {
		return nil
	}
	return &slackNotifier{webhookURL: webhookURL, client: client}
}

func (n *slackNotifier) Name() string { return "slack" }

func (n *slackNotifier) Send(ctx context.Context, msg Message) error {
	body := map[string]any{"text": slackText(msg)}
	return postJSON(ctx, n.client, n.webhookURL, body, "slack")
}

type teamsNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewTeamsNotifier(webhookURL string, client *http.Client) Notifier {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" || client == nil {
		return nil
	}
	return &teamsNotifier{webhookURL: webhookURL, client: client}
}

func (n *teamsNotifier) Name() string { return "teams" }

func (n *teamsNotifier) Send(ctx context.Context, msg Message) error {
	body := map[string]any{
		"@type":    "MessageCard",
		"@context": "http://schema.org/extensions",
		"summary":  msg.Subject,
		"title":    msg.Subject,
		"text":     slackText(msg),
	}
	if len(msg.Fields) > 0 {
		sections := []map[string]any{{"facts": fieldFacts(msg.Fields)}}
		body["sections"] = sections
	}
	return postJSON(ctx, n.client, n.webhookURL, body, "teams")
}

type webhookNotifier struct {
	name       string
	webhookURL string
	client     *http.Client
}

func NewWebhookNotifier(webhookURL string, client *http.Client, name string) Notifier {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" || client == nil {
		return nil
	}
	if name == "" {
		name = "webhook"
	}
	return &webhookNotifier{name: name, webhookURL: webhookURL, client: client}
}

func (n *webhookNotifier) Name() string { return n.name }

func (n *webhookNotifier) Send(ctx context.Context, msg Message) error {
	return postJSON(ctx, n.client, n.webhookURL, msg, n.name)
}

func slackText(msg Message) string {
	var b strings.Builder
	b.WriteString(msg.Subject)
	if msg.Text != "" {
		b.WriteString("\n\n")
		b.WriteString(msg.Text)
	}
	if len(msg.Fields) > 0 {
		b.WriteString("\n\n")
		for k, v := range msg.Fields {
			fmt.Fprintf(&b, "*%s*: %s\n", k, v)
		}
	}
	return strings.TrimSpace(b.String())
}

func fieldFacts(fields map[string]string) []map[string]string {
	out := make([]map[string]string, 0, len(fields))
	for k, v := range fields {
		out = append(out, map[string]string{"name": k, "value": v})
	}
	return out
}

func postJSON(ctx context.Context, client *http.Client, url string, body any, channel string) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned status %d", channel, resp.StatusCode)
	}
	return nil
}

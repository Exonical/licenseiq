package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
)

type fakeNotifier struct {
	name string
	err  error
	seen []Message
}

func (f *fakeNotifier) Name() string { return f.name }
func (f *fakeNotifier) Send(_ context.Context, msg Message) error {
	f.seen = append(f.seen, msg)
	return f.err
}

func TestRenderRenewalReminder(t *testing.T) {
	msg, err := RenderRenewalReminder(RenewalReminderData{
		VendorName:  "Acme",
		ProductName: "LicenseIQ",
		LicenseName: "Annual license",
		RenewalDate: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		DaysUntil:   30,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(msg.Subject, "30 days") {
		t.Fatalf("unexpected subject: %s", msg.Subject)
	}
	if !strings.Contains(msg.Text, "Acme") || !strings.Contains(msg.Text, "2026-01-31") {
		t.Fatalf("unexpected text: %s", msg.Text)
	}
	if !strings.Contains(msg.HTML, "<strong>LicenseIQ</strong>") {
		t.Fatalf("unexpected html: %s", msg.HTML)
	}
}

func TestBuildSMTPMessage(t *testing.T) {
	msg := Message{Subject: "hello", Text: "plain", HTML: "<p>plain</p>"}
	data, err := buildSMTPMessage("from@example.com", []string{"to@example.com"}, msg)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	body := string(data)
	for _, want := range []string{"Subject: hello", "From: from@example.com", "To: to@example.com", "multipart/alternative", "text/plain; charset=utf-8", "text/html; charset=utf-8"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body: %s", want, body)
		}
	}
}

func TestHTTPNotifiers(t *testing.T) {
	tests := []struct {
		name string
		new  func(string, *http.Client) Notifier
	}{
		{name: "slack", new: NewSlackNotifier},
		{name: "teams", new: NewTeamsNotifier},
		{name: "webhook", new: func(url string, client *http.Client) Notifier { return NewWebhookNotifier(url, client, "webhook") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotContentType string
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotContentType = r.Header.Get("Content-Type")
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode: %v", err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			notifier := tc.new(server.URL, server.Client())
			if notifier == nil {
				t.Fatalf("expected notifier")
			}
			msg := Message{Subject: "subject", Text: "text", Fields: map[string]string{"a": "b"}}
			if err := notifier.Send(context.Background(), msg); err != nil {
				t.Fatalf("send: %v", err)
			}
			if gotMethod != http.MethodPost {
				t.Fatalf("expected POST, got %s", gotMethod)
			}
			if gotContentType != "application/json" {
				t.Fatalf("expected application/json, got %s", gotContentType)
			}
			if tc.name == "webhook" {
				if gotBody["subject"] != "subject" {
					t.Fatalf("unexpected webhook body: %+v", gotBody)
				}
				return
			}
			if tc.name == "slack" {
				if gotBody["text"] == "" {
					t.Fatalf("expected slack text payload: %+v", gotBody)
				}
			}
			if tc.name == "teams" {
				if gotBody["@type"] != "MessageCard" {
					t.Fatalf("unexpected teams payload: %+v", gotBody)
				}
			}
		})
	}
}

func TestHTTPNotifierErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()
	notifier := NewSlackNotifier(server.URL, server.Client())
	if notifier == nil {
		t.Fatal("expected notifier")
	}
	if err := notifier.Send(context.Background(), Message{Subject: "s"}); err == nil || !strings.Contains(err.Error(), "slack returned status 418") {
		t.Fatalf("expected non-2xx error, got %v", err)
	}
}

func TestDispatcherFanOutAndNoOp(t *testing.T) {
	dispatcher := &Dispatcher{channels: []Notifier{
		&fakeNotifier{name: "a"},
		&fakeNotifier{name: "b", err: context.Canceled},
	}}
	results := dispatcher.Dispatch(context.Background(), Message{Subject: "subject"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Channel != "a" || !results[0].Success {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].Channel != "b" || results[1].Success || results[1].Error == "" {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
	if err := dispatcher.Send(context.Background(), Message{Subject: "subject"}); err == nil {
		t.Fatalf("expected aggregated error")
	}
	if got := (&Dispatcher{}).Dispatch(context.Background(), Message{}); got != nil {
		t.Fatalf("expected no-op dispatch, got %+v", got)
	}
}

func TestNewDispatcherDisabledByDefault(t *testing.T) {
	dispatcher, err := NewDispatcher(config.NotificationsConfig{})
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if !dispatcher.Empty() {
		t.Fatalf("expected empty dispatcher")
	}
	if err := dispatcher.Send(context.Background(), Message{Subject: "noop"}); err != nil {
		t.Fatalf("expected no-op send, got %v", err)
	}
}

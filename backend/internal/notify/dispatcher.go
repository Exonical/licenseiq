package notify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
)

type Dispatcher struct {
	channels []Notifier
}

func NewDispatcher(cfg config.NotificationsConfig) (*Dispatcher, error) {
	client := &http.Client{Timeout: cfg.HTTPTimeout}
	channels := make([]Notifier, 0, 8)

	if n, err := NewSMTPNotifier(cfg.SMTP); err != nil {
		return nil, err
	} else if n != nil {
		channels = append(channels, n)
	}
	if n := NewSlackNotifier(cfg.Slack.WebhookURL, client); n != nil {
		channels = append(channels, n)
	}
	if n := NewTeamsNotifier(cfg.Teams.WebhookURL, client); n != nil {
		channels = append(channels, n)
	}
	for i, u := range cfg.Webhooks.URLs {
		if n := NewWebhookNotifier(strings.TrimSpace(u), client, fmt.Sprintf("webhook[%d]", i+1)); n != nil {
			channels = append(channels, n)
		}
	}
	return &Dispatcher{channels: channels}, nil
}

func (d *Dispatcher) Name() string { return "dispatcher" }

func (d *Dispatcher) Send(ctx context.Context, msg Message) error {
	results := d.Dispatch(ctx, msg)
	var errs []error
	for _, result := range results {
		if !result.Success && result.Error != "" {
			errs = append(errs, errors.New(result.Channel+": "+result.Error))
		}
	}
	return errors.Join(errs...)
}

func (d *Dispatcher) Dispatch(ctx context.Context, msg Message) []Result {
	if d == nil || len(d.channels) == 0 {
		return nil
	}
	results := make([]Result, 0, len(d.channels))
	for _, channel := range d.channels {
		err := channel.Send(ctx, msg)
		result := Result{Channel: channel.Name(), Success: err == nil}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results
}

func (d *Dispatcher) Channels() []Notifier {
	if d == nil {
		return nil
	}
	return append([]Notifier(nil), d.channels...)
}

func (d *Dispatcher) Empty() bool {
	return d == nil || len(d.channels) == 0
}

func TestMessage() Message {
	return Message{
		Subject: "LicenseIQ notification test",
		Text:    "This is a test notification from LicenseIQ.",
		HTML:    "<p>This is a test notification from LicenseIQ.</p>",
		Fields: map[string]string{
			"service": "licenseiq",
			"kind":    "test",
		},
	}
}

func SafeDispatchTimeout() time.Duration { return 10 * time.Second }

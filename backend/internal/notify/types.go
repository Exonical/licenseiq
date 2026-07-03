package notify

import "context"

type Message struct {
	Subject string            `json:"subject"`
	Text    string            `json:"text"`
	HTML    string            `json:"html,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type Result struct {
	Channel string `json:"channel"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type Notifier interface {
	Name() string
	Send(context.Context, Message) error
}

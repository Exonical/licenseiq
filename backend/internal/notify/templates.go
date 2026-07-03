package notify

import (
	"bytes"
	"fmt"
	"html/template"
	texttmpl "text/template"
	"time"
)

type RenewalReminderData struct {
	VendorName  string
	ProductName string
	LicenseName string
	RenewalDate time.Time
	DaysUntil   int
}

func RenderRenewalReminder(data RenewalReminderData) (Message, error) {
	subject, err := renderTextTemplate(`Renewal reminder: {{if .ProductName}}{{.ProductName}}{{else}}{{.LicenseName}}{{end}} renews in {{.DaysUntil}} days`, data)
	if err != nil {
		return Message{}, err
	}
	textBody, err := renderTextTemplate(`{{if .LicenseName}}License: {{.LicenseName}}{{"\n"}}{{end}}{{if .VendorName}}Vendor: {{.VendorName}}{{"\n"}}{{end}}{{if .ProductName}}Product: {{.ProductName}}{{"\n"}}{{end}}Renewal date: {{.RenewalDate.Format "2006-01-02"}}{{"\n"}}Days remaining: {{.DaysUntil}}`, data)
	if err != nil {
		return Message{}, err
	}
	htmlBody, err := renderHTMLTemplate(`<!doctype html><html><body><h1>Renewal reminder</h1><p>{{if .LicenseName}}License: <strong>{{.LicenseName}}</strong><br>{{end}}{{if .VendorName}}Vendor: <strong>{{.VendorName}}</strong><br>{{end}}{{if .ProductName}}Product: <strong>{{.ProductName}}</strong><br>{{end}}Renewal date: <strong>{{.RenewalDate.Format "2006-01-02"}}</strong><br>Days remaining: <strong>{{.DaysUntil}}</strong></p></body></html>`, data)
	if err != nil {
		return Message{}, err
	}
	fields := map[string]string{
		"daysUntil":   fmt.Sprintf("%d", data.DaysUntil),
		"renewalDate": data.RenewalDate.Format(time.RFC3339),
	}
	if data.VendorName != "" {
		fields["vendorName"] = data.VendorName
	}
	if data.ProductName != "" {
		fields["productName"] = data.ProductName
	}
	if data.LicenseName != "" {
		fields["licenseName"] = data.LicenseName
	}
	return Message{Subject: subject, Text: textBody, HTML: htmlBody, Fields: fields}, nil
}

func renderTextTemplate(tpl string, data RenewalReminderData) (string, error) {
	t, err := texttmpl.New("message").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderHTMLTemplate(tpl string, data RenewalReminderData) (string, error) {
	t, err := template.New("message").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

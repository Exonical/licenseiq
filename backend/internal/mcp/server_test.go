package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"
)

func TestServerListsToolsResourcesAndPrompts(t *testing.T) {
	session, cleanup, deps, _ := startTestSession(t, true)
	defer cleanup()

	tools, err := session.ListTools(context.Background(), &sdkmcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]*sdkmcp.Tool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = tool
	}
	expected := []string{
		"search-licenses", "create-license", "update-license", "renew-license",
		"search-vendors", "create-vendor", "generate-report", "query-dashboard",
		"create-jira-issue",
	}
	for _, name := range expected {
		if _, ok := names[name]; !ok {
			t.Fatalf("missing tool %q", name)
		}
		if names[name].InputSchema == nil {
			t.Fatalf("tool %q missing input schema", name)
		}
	}
	props := schemaProperties(t, names["create-license"].InputSchema)
	for _, want := range []string{"vendor_id", "product_id", "type"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("create-license schema missing %q", want)
		}
	}

	resources, err := session.ListResources(context.Background(), &sdkmcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(resources.Resources) != 1 || resources.Resources[0].URI != defaultDashboardResource {
		t.Fatalf("unexpected resources: %#v", resources.Resources)
	}
	resource, err := session.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{URI: defaultDashboardResource})
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if len(resource.Contents) != 1 {
		t.Fatalf("expected 1 resource content, got %d", len(resource.Contents))
	}
	var dashboard dashboardSummary
	if err := json.Unmarshal([]byte(resource.Contents[0].Text), &dashboard); err != nil {
		t.Fatalf("decode dashboard resource: %v", err)
	}
	if dashboard.Licenses == 0 || dashboard.Vendors == 0 {
		t.Fatalf("unexpected dashboard summary: %#v", dashboard)
	}

	prompts, err := session.ListPrompts(context.Background(), &sdkmcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(prompts.Prompts) != 1 || prompts.Prompts[0].Name != defaultPromptName {
		t.Fatalf("unexpected prompts: %#v", prompts.Prompts)
	}
	prompt, err := session.GetPrompt(context.Background(), &sdkmcp.GetPromptParams{
		Name:      defaultPromptName,
		Arguments: map[string]string{"window_days": "45", "as_of": "2026-01-02T00:00:00Z"},
	})
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if len(prompt.Messages) != 1 {
		t.Fatalf("expected 1 prompt message, got %d", len(prompt.Messages))
	}
	text := promptMessageText(t, prompt.Messages[0])
	if !strings.Contains(text, "45 days") || !strings.Contains(text, defaultDashboardResource) {
		t.Fatalf("unexpected prompt text: %s", text)
	}

	_ = deps
}

func TestCreateLicenseToolWritesAuditAndUsesActorContext(t *testing.T) {
	session, cleanup, deps, auditRepo := startTestSession(t, true)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "create-license",
		Arguments: map[string]any{
			"vendor_id":      deps.vendorID.String(),
			"product_id":     deps.productID.String(),
			"license_key":    "NEW-100",
			"seat_count":     20,
			"assigned_seats": 4,
			"cost":           "199.99",
			"currency":       "USD",
			"type":           string(domain.LicenseTypeSubscription),
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", contentText(t, result.Content[0]))
	}
	license := decodeStructured[licenseRecord](t, result.StructuredContent)
	if license.LicenseKey != "NEW-100" || license.VendorID != deps.vendorID.String() {
		t.Fatalf("unexpected license result: %#v", license)
	}
	if len(auditRepo.logs) == 0 {
		t.Fatal("expected audit entry")
	}
	last := auditRepo.logs[len(auditRepo.logs)-1]
	if last.ActorUserID == nil || *last.ActorUserID != deps.actorID {
		t.Fatalf("unexpected actor on audit log: %#v", last.ActorUserID)
	}
	if last.EntityType != "license" {
		t.Fatalf("unexpected audit entity type: %s", last.EntityType)
	}
}

func TestGenerateReportAndDisabledJiraTool(t *testing.T) {
	session, cleanup, deps, _ := startTestSession(t, true)
	defer cleanup()

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "generate-report",
		Arguments: map[string]any{
			"report_type": "upcoming-renewals",
			"window_days": 30,
			"as_of":       deps.asOf.Format(time.RFC3339),
		},
	})
	if err != nil {
		t.Fatalf("call report tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected report error: %v", result.GetError())
	}
	report := decodeStructured[reportOutput](t, result.StructuredContent)
	if report.Report.Title == "" || len(report.Report.Rows) == 0 {
		t.Fatalf("unexpected report: %#v", report.Report)
	}

	disabledSession, cleanupDisabled, _, _ := startTestSession(t, false)
	defer cleanupDisabled()
	disabledResult, err := disabledSession.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "create-jira-issue",
		Arguments: map[string]any{"license_id": deps.licenseID.String()},
	})
	if err != nil {
		t.Fatalf("call disabled jira tool: %v", err)
	}
	if !disabledResult.IsError {
		t.Fatal("expected jira tool to return an error result")
	}
	if len(disabledResult.Content) == 0 {
		t.Fatal("expected jira error content")
	}
	if text := contentText(t, disabledResult.Content[0]); !strings.Contains(text, "jira is not configured") {
		t.Fatalf("unexpected jira error: %s", text)
	}
}

func TestInvalidInputReturnsToolError(t *testing.T) {
	session, cleanup, _, _ := startTestSession(t, true)
	defer cleanup()
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "create-license",
		Arguments: map[string]any{
			"vendor_id":  "",
			"product_id": "",
			"type":       string(domain.LicenseTypeSubscription),
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected error details")
	}
	t.Logf("invalid input error: %s", contentText(t, result.Content[0]))
}

func startTestSession(t *testing.T, jiraEnabled bool) (*sdkmcp.ClientSession, func(), testDeps, *auditRepoFake) {
	t.Helper()
	vendorID := uuid.New()
	productID := uuid.New()
	licenseID := uuid.New()
	actorID := uuid.New()
	now := time.Now().UTC()

	auditRepo := &auditRepoFake{}
	vendorRepo := &vendorRepoFake{items: map[uuid.UUID]domain.Vendor{
		vendorID: {Base: domain.Base{ID: vendorID, CreatedAt: now, UpdatedAt: now}, Name: "Acme"},
	}}
	productRepo := &productRepoFake{items: map[uuid.UUID]domain.Product{
		productID: {Base: domain.Base{ID: productID, CreatedAt: now, UpdatedAt: now}, Name: "Widget"},
	}}
	licenseRepo := &licenseRepoFake{items: map[uuid.UUID]domain.License{
		licenseID: {
			Base:                  domain.Base{ID: licenseID, CreatedAt: now, UpdatedAt: now},
			VendorID:              vendorID,
			ProductID:             productID,
			LicenseKey:            "LIC-001",
			RenewalDate:           timePtr(now.AddDate(0, 0, 15)),
			ExpirationDate:        timePtr(now.AddDate(0, 0, 90)),
			MaintenanceExpiration: timePtr(now.AddDate(0, 0, 90)),
			SeatCount:             10,
			AssignedSeats:         4,
			Cost:                  decimal.RequireFromString("100.00"),
			Currency:              "USD",
			Type:                  domain.LicenseTypeSubscription,
		},
	}}
	assignmentRepo := &assignmentRepoFake{}
	licenseSvc := app.NewLicenseService(licenseRepo, auditRepo)
	vendorSvc := app.NewVendorService(vendorRepo, auditRepo)
	productSvc := app.NewProductService(productRepo, auditRepo)
	reportSvc := app.NewReportingService(vendorRepo, productRepo, licenseRepo)
	var jiraSvc app.JiraService
	if jiraEnabled {
		jiraSvc = &jiraSvcFake{}
	}
	server := NewServer(Dependencies{
		Licenses:    licenseSvc,
		Vendors:     vendorSvc,
		Products:    productSvc,
		Assignments: app.NewAssignmentService(assignmentRepo, auditRepo),
		Reports:     reportSvc,
		Jira:        jiraSvc,
	}, Options{
		Principal: auth.Principal{
			UserID:           &actorID,
			Role:             domain.RoleAdministrator,
			Email:            "mcp@example.com",
			IsServiceAccount: true,
		},
		SessionID:         "test",
		DashboardResource: defaultDashboardResource,
	})

	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- server.Run(ctx, serverTransport) }()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("connect client: %v", err)
	}
	cleanup := func() {
		_ = session.Close()
		cancel()
		select {
		case <-time.After(2 * time.Second):
			t.Fatal("server did not stop")
		case <-errCh:
		}
	}
	return session, cleanup, testDeps{
		vendorID:  vendorID,
		productID: productID,
		licenseID: licenseID,
		actorID:   actorID,
		asOf:      now,
	}, auditRepo
}

type testDeps struct {
	vendorID  uuid.UUID
	productID uuid.UUID
	licenseID uuid.UUID
	actorID   uuid.UUID
	asOf      time.Time
}

func schemaProperties(t *testing.T, schema any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	props, ok := decoded["properties"].(map[string]any)
	if !ok {
		t.Fatalf("missing properties: %#v", decoded)
	}
	return props
}

func decodeStructured[T any](t *testing.T, value any) T {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	return out
}

func promptMessageText(t *testing.T, message *sdkmcp.PromptMessage) string {
	t.Helper()
	text, ok := message.Content.(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("unexpected prompt content: %#v", message.Content)
	}
	return text.Text
}

func contentText(t *testing.T, content sdkmcp.Content) string {
	t.Helper()
	text, ok := content.(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %#v", content)
	}
	return text.Text
}

func timePtr(t time.Time) *time.Time { return &t }

type auditRepoFake struct {
	logs []domain.AuditLog
}

func (r *auditRepoFake) Create(_ context.Context, log *domain.AuditLog) error {
	r.logs = append(r.logs, *log)
	return nil
}
func (r *auditRepoFake) Get(context.Context, uuid.UUID) (*domain.AuditLog, error) {
	return nil, domain.ErrNotFound
}
func (r *auditRepoFake) List(context.Context, domain.ListFilter) ([]domain.AuditLog, error) {
	return append([]domain.AuditLog(nil), r.logs...), nil
}

type vendorRepoFake struct {
	items map[uuid.UUID]domain.Vendor
}

func (r *vendorRepoFake) Create(_ context.Context, v *domain.Vendor) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	v.CreatedAt = now
	v.UpdatedAt = now
	r.items[v.ID] = *v
	return nil
}
func (r *vendorRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Vendor, error) {
	v, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &v, nil
}
func (r *vendorRepoFake) Update(_ context.Context, v *domain.Vendor) error {
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	v.UpdatedAt = now
	r.items[v.ID] = *v
	return nil
}
func (r *vendorRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *vendorRepoFake) List(context.Context, domain.ListFilter) ([]domain.Vendor, error) {
	out := make([]domain.Vendor, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

type productRepoFake struct {
	items map[uuid.UUID]domain.Product
}

func (r *productRepoFake) Create(_ context.Context, p *domain.Product) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	p.CreatedAt = now
	p.UpdatedAt = now
	r.items[p.ID] = *p
	return nil
}
func (r *productRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	p, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &p, nil
}
func (r *productRepoFake) Update(_ context.Context, p *domain.Product) error {
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = now
	r.items[p.ID] = *p
	return nil
}
func (r *productRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *productRepoFake) List(context.Context, domain.ListFilter) ([]domain.Product, error) {
	out := make([]domain.Product, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

type licenseRepoFake struct {
	items map[uuid.UUID]domain.License
}

func (r *licenseRepoFake) Create(_ context.Context, l *domain.License) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	l.CreatedAt = now
	l.UpdatedAt = now
	r.items[l.ID] = *l
	return nil
}
func (r *licenseRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.License, error) {
	l, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &l, nil
}
func (r *licenseRepoFake) Update(_ context.Context, l *domain.License) error {
	now := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	l.UpdatedAt = now
	if created, ok := r.items[l.ID]; ok {
		l.CreatedAt = created.CreatedAt
	}
	r.items[l.ID] = *l
	return nil
}
func (r *licenseRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *licenseRepoFake) List(context.Context, domain.ListFilter) ([]domain.License, error) {
	out := make([]domain.License, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

type assignmentRepoFake struct {
	items map[uuid.UUID]domain.Assignment
}

func (r *assignmentRepoFake) Create(_ context.Context, a *domain.Assignment) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	r.items[a.ID] = *a
	return nil
}
func (r *assignmentRepoFake) Get(_ context.Context, id uuid.UUID) (*domain.Assignment, error) {
	a, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &a, nil
}
func (r *assignmentRepoFake) Update(_ context.Context, a *domain.Assignment) error {
	r.items[a.ID] = *a
	return nil
}
func (r *assignmentRepoFake) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *assignmentRepoFake) List(context.Context, domain.ListFilter) ([]domain.Assignment, error) {
	out := make([]domain.Assignment, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

type jiraSvcFake struct {
	calls []uuid.UUID
}

func (j *jiraSvcFake) CreateRenewalTicket(_ context.Context, licenseID uuid.UUID) (*domain.LicenseIssueLink, error) {
	j.calls = append(j.calls, licenseID)
	return &domain.LicenseIssueLink{
		Base:      domain.Base{ID: uuid.New(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		LicenseID: licenseID,
		IssueKey:  "JIRA-123",
		IssueURL:  "https://jira.example.com/browse/JIRA-123",
		Status:    "Open",
	}, nil
}
func (j *jiraSvcFake) LinkIssue(context.Context, uuid.UUID, string, string) (*domain.LicenseIssueLink, error) {
	return nil, errors.New("not implemented")
}
func (j *jiraSvcFake) ListIssueLinks(context.Context, uuid.UUID) ([]domain.LicenseIssueLink, error) {
	return nil, nil
}
func (j *jiraSvcFake) UpdateIssueStatus(context.Context, uuid.UUID, string, string) (*domain.LicenseIssueLink, error) {
	return nil, errors.New("not implemented")
}
func (j *jiraSvcFake) AttachIssueFile(context.Context, uuid.UUID, string, uuid.UUID) error {
	return errors.New("not implemented")
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/app"
	"github.com/Exonical/licenseiq/backend/internal/auth"
	"github.com/Exonical/licenseiq/backend/internal/domain"
	"github.com/Exonical/licenseiq/backend/internal/reporting"
	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"
)

const (
	defaultSessionID         = "mcp"
	defaultDashboardResource = "licenseiq://dashboard/summary"
	defaultPromptName        = "renewal-review"
)

type Dependencies struct {
	Licenses    app.LicenseService
	Vendors     app.VendorService
	Products    app.ProductService
	Assignments app.AssignmentService
	Reports     app.ReportingService
	Jira        app.JiraService
}

type Options struct {
	Principal         auth.Principal
	SessionID         string
	DashboardResource string
}

type serverState struct {
	deps              Dependencies
	principal         auth.Principal
	sessionID         string
	dashboardResource string
}

type LicenseWriteInput struct {
	VendorID              string     `json:"vendor_id"`
	ProductID             string     `json:"product_id"`
	Department            string     `json:"department,omitempty"`
	LicenseKey            string     `json:"license_key,omitempty"`
	SubscriptionID        string     `json:"subscription_id,omitempty"`
	ContractNumber        string     `json:"contract_number,omitempty"`
	PurchaseOrder         string     `json:"purchase_order,omitempty"`
	Invoice               string     `json:"invoice,omitempty"`
	PurchaseDate          *time.Time `json:"purchase_date,omitempty"`
	RenewalDate           *time.Time `json:"renewal_date,omitempty"`
	ExpirationDate        *time.Time `json:"expiration_date,omitempty"`
	MaintenanceExpiration *time.Time `json:"maintenance_expiration,omitempty"`
	SeatCount             int        `json:"seat_count,omitempty"`
	AssignedSeats         int        `json:"assigned_seats,omitempty"`
	Cost                  string     `json:"cost,omitempty"`
	Currency              string     `json:"currency,omitempty"`
	Notes                 string     `json:"notes,omitempty"`
	Type                  string     `json:"type"`
}

type CreateLicenseInput struct {
	LicenseWriteInput
}

type UpdateLicenseInput struct {
	LicenseID string `json:"license_id"`
	LicenseWriteInput
}

type RenewLicenseInput struct {
	LicenseID   string    `json:"license_id"`
	RenewalDate time.Time `json:"renewal_date"`
}

type SearchLicensesInput struct {
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type licenseRecord struct {
	ID                    string     `json:"id"`
	VendorID              string     `json:"vendor_id"`
	VendorName            string     `json:"vendor_name,omitempty"`
	ProductID             string     `json:"product_id"`
	ProductName           string     `json:"product_name,omitempty"`
	Department            string     `json:"department,omitempty"`
	LicenseKey            string     `json:"license_key,omitempty"`
	SubscriptionID        string     `json:"subscription_id,omitempty"`
	ContractNumber        string     `json:"contract_number,omitempty"`
	PurchaseOrder         string     `json:"purchase_order,omitempty"`
	Invoice               string     `json:"invoice,omitempty"`
	PurchaseDate          *time.Time `json:"purchase_date,omitempty"`
	RenewalDate           *time.Time `json:"renewal_date,omitempty"`
	ExpirationDate        *time.Time `json:"expiration_date,omitempty"`
	MaintenanceExpiration *time.Time `json:"maintenance_expiration,omitempty"`
	SeatCount             int        `json:"seat_count"`
	AssignedSeats         int        `json:"assigned_seats"`
	AvailableSeats        int        `json:"available_seats"`
	Cost                  string     `json:"cost,omitempty"`
	Currency              string     `json:"currency,omitempty"`
	Notes                 string     `json:"notes,omitempty"`
	Type                  string     `json:"type"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type licenseSearchResult struct {
	Count    int             `json:"count"`
	Licenses []licenseRecord `json:"licenses"`
}

type VendorContactInput struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
	Role  string `json:"role,omitempty"`
}

type VendorWriteInput struct {
	Name           string               `json:"name"`
	SupportURL     string               `json:"support_url,omitempty"`
	AccountManager string               `json:"account_manager,omitempty"`
	Notes          string               `json:"notes,omitempty"`
	Contacts       []VendorContactInput `json:"contacts,omitempty"`
}

type CreateVendorInput struct {
	VendorWriteInput
}

type UpdateVendorInput struct {
	VendorID string `json:"vendor_id"`
	VendorWriteInput
}

type vendorRecord struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	SupportURL     string               `json:"support_url,omitempty"`
	AccountManager string               `json:"account_manager,omitempty"`
	Notes          string               `json:"notes,omitempty"`
	Contacts       []VendorContactInput `json:"contacts,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type SearchVendorInput struct {
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type vendorSearchResult struct {
	Count   int            `json:"count"`
	Vendors []vendorRecord `json:"vendors"`
}

type ReportInput struct {
	ReportType string     `json:"report_type"`
	AsOf       *time.Time `json:"as_of,omitempty"`
	WindowDays int        `json:"window_days,omitempty"`
}

type reportOutput struct {
	Report reporting.Table `json:"report"`
}

type DashboardInput struct {
	AsOf *time.Time `json:"as_of,omitempty"`
}

type dashboardSummary struct {
	AsOf             time.Time `json:"as_of"`
	Licenses         int       `json:"licenses"`
	Vendors          int       `json:"vendors"`
	Products         int       `json:"products"`
	Assignments      int       `json:"assignments"`
	UpcomingRenewals int       `json:"upcoming_renewals"`
	ExpiredLicenses  int       `json:"expired_licenses"`
	SeatCount        int       `json:"seat_count"`
	AssignedSeats    int       `json:"assigned_seats"`
	AvailableSeats   int       `json:"available_seats"`
}

type dashboardOutput struct {
	Dashboard dashboardSummary `json:"dashboard"`
}

type JiraIssueInput struct {
	LicenseID string `json:"license_id"`
}

type jiraIssueOutput struct {
	Link jiraIssueLinkRecord `json:"link"`
}

type jiraIssueLinkRecord struct {
	ID          string     `json:"id"`
	LicenseID   string     `json:"license_id"`
	IssueKey    string     `json:"issue_key"`
	IssueURL    string     `json:"issue_url,omitempty"`
	Status      string     `json:"status,omitempty"`
	RenewalDate *time.Time `json:"renewal_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewServer(deps Dependencies, opts Options) *sdkmcp.Server {
	if opts.SessionID == "" {
		opts.SessionID = defaultSessionID
	}
	if opts.DashboardResource == "" {
		opts.DashboardResource = defaultDashboardResource
	}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "licenseiq", Version: "dev"}, &sdkmcp.ServerOptions{
		Instructions: "LicenseIQ automation server. Use the tools to search and manage licenses, vendors, reports, dashboard summaries, and Jira renewal tickets.",
	})
	state := &serverState{
		deps:              deps,
		principal:         opts.Principal,
		sessionID:         opts.SessionID,
		dashboardResource: opts.DashboardResource,
	}
	state.register(server)
	return server
}

func (s *serverState) register(server *sdkmcp.Server) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search-licenses",
		Description: "Search licenses by free-text query.",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, Title: "Search Licenses"},
	}, s.searchLicenses)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create-license",
		Description: "Create a license.",
		Annotations: &sdkmcp.ToolAnnotations{DestructiveHint: boolPtr(true), Title: "Create License"},
	}, s.createLicense)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "update-license",
		Description: "Update a license.",
		Annotations: &sdkmcp.ToolAnnotations{DestructiveHint: boolPtr(true), Title: "Update License"},
	}, s.updateLicense)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "renew-license",
		Description: "Set the renewal date for a license.",
		Annotations: &sdkmcp.ToolAnnotations{DestructiveHint: boolPtr(true), Title: "Renew License"},
	}, s.renewLicense)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search-vendors",
		Description: "Search vendors by free-text query.",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, Title: "Search Vendors"},
	}, s.searchVendors)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create-vendor",
		Description: "Create a vendor.",
		Annotations: &sdkmcp.ToolAnnotations{DestructiveHint: boolPtr(true), Title: "Create Vendor"},
	}, s.createVendor)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "generate-report",
		Description: "Generate a LicenseIQ report.",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, Title: "Generate Report"},
	}, s.generateReport)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "query-dashboard",
		Description: "Query a dashboard summary derived from the current data.",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, Title: "Query Dashboard"},
	}, s.queryDashboard)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create-jira-issue",
		Description: "Create a Jira renewal ticket for a license.",
		Annotations: &sdkmcp.ToolAnnotations{DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(true), Title: "Create Jira Issue"},
	}, s.createJiraIssue)

	server.AddResource(&sdkmcp.Resource{
		Name:        "dashboard-summary",
		Title:       "Dashboard Summary",
		Description: "A JSON dashboard summary derived from LicenseIQ data.",
		URI:         s.dashboardResource,
		MIMEType:    "application/json",
	}, s.readDashboardSummary)

	server.AddPrompt(&sdkmcp.Prompt{
		Name:        defaultPromptName,
		Title:       "Renewal Review",
		Description: "A prompt for reviewing upcoming renewals.",
		Arguments: []*sdkmcp.PromptArgument{
			{Name: "window_days", Title: "Window Days", Description: "Number of days to inspect", Required: false},
			{Name: "as_of", Title: "As Of", Description: "RFC3339 timestamp to use as the reference date", Required: false},
		},
	}, s.renewalReviewPrompt)
}

func (s *serverState) searchLicenses(ctx context.Context, _ *sdkmcp.CallToolRequest, input SearchLicensesInput) (*sdkmcp.CallToolResult, licenseSearchResult, error) {
	licenses, err := listLicenses(ctx, s.deps.Licenses)
	if err != nil {
		return nil, licenseSearchResult{}, err
	}
	vendors, products, err := s.catalog(ctx)
	if err != nil {
		return nil, licenseSearchResult{}, err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	records := make([]licenseRecord, 0, len(licenses))
	for _, lic := range licenses {
		record := licenseRecordFromDomain(lic, vendors[lic.VendorID], products[lic.ProductID])
		if query != "" && !licenseMatches(record, query) {
			continue
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return licenseSortKey(records[i]) < licenseSortKey(records[j])
	})
	total := len(records)
	records = paginate(records, input.Limit, input.Offset)
	return nil, licenseSearchResult{Count: total, Licenses: records}, nil
}

func (s *serverState) createLicense(ctx context.Context, _ *sdkmcp.CallToolRequest, input CreateLicenseInput) (*sdkmcp.CallToolResult, licenseRecord, error) {
	ctx = s.toolContext(ctx)
	lic, err := s.writeLicense(ctx, input.LicenseWriteInput, nil)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	vendors, products, err := s.catalog(ctx)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	return nil, licenseRecordFromDomain(*lic, vendors[lic.VendorID], products[lic.ProductID]), nil
}

func (s *serverState) updateLicense(ctx context.Context, _ *sdkmcp.CallToolRequest, input UpdateLicenseInput) (*sdkmcp.CallToolResult, licenseRecord, error) {
	ctx = s.toolContext(ctx)
	licenseID, err := parseUUID(input.LicenseID)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	lic, err := s.writeLicense(ctx, input.LicenseWriteInput, &licenseID)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	vendors, products, err := s.catalog(ctx)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	return nil, licenseRecordFromDomain(*lic, vendors[lic.VendorID], products[lic.ProductID]), nil
}

func (s *serverState) renewLicense(ctx context.Context, _ *sdkmcp.CallToolRequest, input RenewLicenseInput) (*sdkmcp.CallToolResult, licenseRecord, error) {
	ctx = s.toolContext(ctx)
	licenseID, err := parseUUID(input.LicenseID)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	license, err := s.deps.Licenses.Get(ctx, licenseID)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	license.RenewalDate = cloneTime(&input.RenewalDate)
	updated, err := s.deps.Licenses.Update(ctx, licenseID, *license)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	vendors, products, err := s.catalog(ctx)
	if err != nil {
		return nil, licenseRecord{}, err
	}
	return nil, licenseRecordFromDomain(*updated, vendors[updated.VendorID], products[updated.ProductID]), nil
}

func (s *serverState) searchVendors(ctx context.Context, _ *sdkmcp.CallToolRequest, input SearchVendorInput) (*sdkmcp.CallToolResult, vendorSearchResult, error) {
	vendors, err := listVendors(ctx, s.deps.Vendors)
	if err != nil {
		return nil, vendorSearchResult{}, err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	records := make([]vendorRecord, 0, len(vendors))
	for _, vendor := range vendors {
		record := vendorRecordFromDomain(vendor)
		if query != "" && !vendorMatches(record, query) {
			continue
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return strings.ToLower(records[i].Name) < strings.ToLower(records[j].Name)
	})
	records = paginate(records, input.Limit, input.Offset)
	return nil, vendorSearchResult{Count: len(records), Vendors: records}, nil
}

func (s *serverState) createVendor(ctx context.Context, _ *sdkmcp.CallToolRequest, input CreateVendorInput) (*sdkmcp.CallToolResult, vendorRecord, error) {
	ctx = s.toolContext(ctx)
	vendor, err := s.writeVendor(ctx, input.VendorWriteInput, nil)
	if err != nil {
		return nil, vendorRecord{}, err
	}
	return nil, vendorRecordFromDomain(*vendor), nil
}

func (s *serverState) generateReport(ctx context.Context, _ *sdkmcp.CallToolRequest, input ReportInput) (*sdkmcp.CallToolResult, reportOutput, error) {
	table, err := s.report(ctx, input)
	if err != nil {
		return nil, reportOutput{}, err
	}
	return nil, reportOutput{Report: table}, nil
}

func (s *serverState) queryDashboard(ctx context.Context, _ *sdkmcp.CallToolRequest, input DashboardInput) (*sdkmcp.CallToolResult, dashboardOutput, error) {
	ctx = s.toolContext(ctx)
	summary, err := s.dashboard(ctx, input.AsOf)
	if err != nil {
		return nil, dashboardOutput{}, err
	}
	return nil, dashboardOutput{Dashboard: summary}, nil
}

func (s *serverState) createJiraIssue(ctx context.Context, _ *sdkmcp.CallToolRequest, input JiraIssueInput) (*sdkmcp.CallToolResult, jiraIssueOutput, error) {
	ctx = s.toolContext(ctx)
	if s.deps.Jira == nil {
		return nil, jiraIssueOutput{}, app.ErrJiraDisabled
	}
	licenseID, err := parseUUID(input.LicenseID)
	if err != nil {
		return nil, jiraIssueOutput{}, err
	}
	link, err := s.deps.Jira.CreateRenewalTicket(ctx, licenseID)
	if err != nil {
		return nil, jiraIssueOutput{}, err
	}
	return nil, jiraIssueOutput{Link: jiraIssueLinkFromDomain(*link)}, nil
}

func (s *serverState) readDashboardSummary(ctx context.Context, _ *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	ctx = s.toolContext(ctx)
	summary, err := s.dashboard(ctx, nil)
	if err != nil {
		return nil, err
	}
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, err
	}
	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{URI: s.dashboardResource, MIMEType: "application/json", Text: string(payload)}},
	}, nil
}

func (s *serverState) renewalReviewPrompt(ctx context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
	windowDays := "90"
	asOf := ""
	if req != nil && req.Params != nil {
		if value := strings.TrimSpace(req.Params.Arguments["window_days"]); value != "" {
			windowDays = value
		}
		asOf = strings.TrimSpace(req.Params.Arguments["as_of"])
	}
	text := fmt.Sprintf("Review upcoming license renewals within %s days%s. Use the dashboard summary resource at %s for context, then identify the licenses that need action and summarize the vendor, product, renewal date, and urgency.", windowDays, promptAsOfSuffix(asOf), s.dashboardResource)
	return &sdkmcp.GetPromptResult{
		Description: "Review upcoming renewals with context from the dashboard summary.",
		Messages: []*sdkmcp.PromptMessage{{
			Role:    "user",
			Content: &sdkmcp.TextContent{Text: text},
		}},
	}, nil
}

func (s *serverState) report(ctx context.Context, input ReportInput) (reporting.Table, error) {
	if s.deps.Reports == nil {
		return reporting.Table{}, fmt.Errorf("reporting service is required")
	}
	asOf := normalizeTime(input.AsOf)
	switch normalizeKey(input.ReportType) {
	case "upcoming-renewals", "renewals":
		table, err := s.deps.Reports.UpcomingRenewals(ctx, app.UpcomingRenewalsParams{AsOf: asOf, WindowDays: input.WindowDays})
		if err != nil {
			return reporting.Table{}, err
		}
		return table, nil
	case "expired-licenses", "expired":
		table, err := s.deps.Reports.ExpiredLicenses(ctx, app.ExpiredLicensesParams{AsOf: asOf})
		if err != nil {
			return reporting.Table{}, err
		}
		return table, nil
	case "vendor-spend":
		table, err := s.deps.Reports.VendorSpend(ctx, app.ReportingAsOfParams{AsOf: asOf})
		if err != nil {
			return reporting.Table{}, err
		}
		return table, nil
	case "license-utilization", "utilization":
		table, err := s.deps.Reports.LicenseUtilization(ctx, app.ReportingAsOfParams{AsOf: asOf})
		if err != nil {
			return reporting.Table{}, err
		}
		return table, nil
	case "department-spend":
		table, err := s.deps.Reports.DepartmentSpend(ctx, app.ReportingAsOfParams{AsOf: asOf})
		if err != nil {
			return reporting.Table{}, err
		}
		return table, nil
	default:
		return reporting.Table{}, fmt.Errorf("unsupported report type %q", input.ReportType)
	}
}

func (s *serverState) dashboard(ctx context.Context, asOf *time.Time) (dashboardSummary, error) {
	if s.deps.Licenses == nil {
		return dashboardSummary{}, fmt.Errorf("license service is required")
	}
	now := normalizeTime(asOf)
	licenses, err := listLicenses(ctx, s.deps.Licenses)
	if err != nil {
		return dashboardSummary{}, err
	}
	vendors, err := listVendors(ctx, s.deps.Vendors)
	if err != nil {
		return dashboardSummary{}, err
	}
	products, err := listProducts(ctx, s.deps.Products)
	if err != nil {
		return dashboardSummary{}, err
	}
	assignments, err := listAssignments(ctx, s.deps.Assignments)
	if err != nil {
		return dashboardSummary{}, err
	}
	upcoming := 0
	if s.deps.Reports != nil {
		if table, err := s.deps.Reports.UpcomingRenewals(ctx, app.UpcomingRenewalsParams{AsOf: now, WindowDays: 90}); err == nil {
			upcoming = len(table.Rows)
		} else {
			return dashboardSummary{}, err
		}
	}
	expired := 0
	if s.deps.Reports != nil {
		if table, err := s.deps.Reports.ExpiredLicenses(ctx, app.ExpiredLicensesParams{AsOf: now}); err == nil {
			expired = len(table.Rows)
		} else {
			return dashboardSummary{}, err
		}
	}
	seatCount := 0
	assignedSeats := 0
	for _, lic := range licenses {
		seatCount += lic.SeatCount
		assignedSeats += lic.AssignedSeats
	}
	availableSeats := seatCount - assignedSeats
	if availableSeats < 0 {
		availableSeats = 0
	}
	return dashboardSummary{
		AsOf:             now,
		Licenses:         len(licenses),
		Vendors:          len(vendors),
		Products:         len(products),
		Assignments:      len(assignments),
		UpcomingRenewals: upcoming,
		ExpiredLicenses:  expired,
		SeatCount:        seatCount,
		AssignedSeats:    assignedSeats,
		AvailableSeats:   availableSeats,
	}, nil
}

func (s *serverState) writeLicense(ctx context.Context, input LicenseWriteInput, id *uuid.UUID) (*domain.License, error) {
	if s.deps.Licenses == nil {
		return nil, fmt.Errorf("license service is required")
	}
	vendorID, err := parseUUID(input.VendorID)
	if err != nil {
		return nil, err
	}
	productID, err := parseUUID(input.ProductID)
	if err != nil {
		return nil, err
	}
	lic := domain.License{
		VendorID:              vendorID,
		ProductID:             productID,
		Department:            input.Department,
		LicenseKey:            input.LicenseKey,
		SubscriptionID:        input.SubscriptionID,
		ContractNumber:        input.ContractNumber,
		PurchaseOrder:         input.PurchaseOrder,
		Invoice:               input.Invoice,
		PurchaseDate:          cloneTime(input.PurchaseDate),
		RenewalDate:           cloneTime(input.RenewalDate),
		ExpirationDate:        cloneTime(input.ExpirationDate),
		MaintenanceExpiration: cloneTime(input.MaintenanceExpiration),
		SeatCount:             input.SeatCount,
		AssignedSeats:         input.AssignedSeats,
		Cost:                  parseDecimal(input.Cost),
		Currency:              input.Currency,
		Notes:                 input.Notes,
		Type:                  domain.LicenseType(input.Type),
	}
	if id == nil {
		created, err := s.deps.Licenses.Create(ctx, lic)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	lic.ID = *id
	updated, err := s.deps.Licenses.Update(ctx, *id, lic)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *serverState) toolContext(ctx context.Context) context.Context {
	ctx = auth.WithPrincipal(ctx, s.principal)
	return app.WithRequestContext(ctx, auth.RequestContext(s.principal, "", s.sessionID))
}

func parseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}

func (s *serverState) writeVendor(ctx context.Context, input VendorWriteInput, id *uuid.UUID) (*domain.Vendor, error) {
	if s.deps.Vendors == nil {
		return nil, fmt.Errorf("vendor service is required")
	}
	vendor := domain.Vendor{
		Name:           input.Name,
		SupportURL:     input.SupportURL,
		AccountManager: input.AccountManager,
		Notes:          input.Notes,
		Contacts:       make([]domain.VendorContact, 0, len(input.Contacts)),
	}
	for _, contact := range input.Contacts {
		vendor.Contacts = append(vendor.Contacts, domain.VendorContact{
			Name:  contact.Name,
			Email: contact.Email,
			Phone: contact.Phone,
			Role:  contact.Role,
		})
	}
	if id == nil {
		created, err := s.deps.Vendors.Create(ctx, vendor)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	vendor.ID = *id
	updated, err := s.deps.Vendors.Update(ctx, *id, vendor)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *serverState) catalog(ctx context.Context) (map[uuid.UUID]string, map[uuid.UUID]string, error) {
	vendors := map[uuid.UUID]string{}
	products := map[uuid.UUID]string{}
	if s.deps.Vendors != nil {
		items, err := listVendors(ctx, s.deps.Vendors)
		if err != nil {
			return nil, nil, err
		}
		for _, vendor := range items {
			vendors[vendor.ID] = vendor.Name
		}
	}
	if s.deps.Products != nil {
		items, err := listProducts(ctx, s.deps.Products)
		if err != nil {
			return nil, nil, err
		}
		for _, product := range items {
			products[product.ID] = product.Name
		}
	}
	return vendors, products, nil
}

func licenseRecordFromDomain(lic domain.License, vendorName, productName string) licenseRecord {
	return licenseRecord{
		ID:                    lic.ID.String(),
		VendorID:              lic.VendorID.String(),
		VendorName:            vendorName,
		ProductID:             lic.ProductID.String(),
		ProductName:           productName,
		Department:            lic.Department,
		LicenseKey:            lic.LicenseKey,
		SubscriptionID:        lic.SubscriptionID,
		ContractNumber:        lic.ContractNumber,
		PurchaseOrder:         lic.PurchaseOrder,
		Invoice:               lic.Invoice,
		PurchaseDate:          cloneTime(lic.PurchaseDate),
		RenewalDate:           cloneTime(lic.RenewalDate),
		ExpirationDate:        cloneTime(lic.ExpirationDate),
		MaintenanceExpiration: cloneTime(lic.MaintenanceExpiration),
		SeatCount:             lic.SeatCount,
		AssignedSeats:         lic.AssignedSeats,
		AvailableSeats:        lic.AvailableSeats(),
		Cost:                  lic.Cost.String(),
		Currency:              lic.Currency,
		Notes:                 lic.Notes,
		Type:                  lic.Type.String(),
		CreatedAt:             lic.CreatedAt,
		UpdatedAt:             lic.UpdatedAt,
	}
}

func vendorRecordFromDomain(vendor domain.Vendor) vendorRecord {
	contacts := make([]VendorContactInput, 0, len(vendor.Contacts))
	for _, contact := range vendor.Contacts {
		contacts = append(contacts, VendorContactInput{
			Name:  contact.Name,
			Email: contact.Email,
			Phone: contact.Phone,
			Role:  contact.Role,
		})
	}
	return vendorRecord{
		ID:             vendor.ID.String(),
		Name:           vendor.Name,
		SupportURL:     vendor.SupportURL,
		AccountManager: vendor.AccountManager,
		Notes:          vendor.Notes,
		Contacts:       contacts,
		CreatedAt:      vendor.CreatedAt,
		UpdatedAt:      vendor.UpdatedAt,
	}
}

func jiraIssueLinkFromDomain(link domain.LicenseIssueLink) jiraIssueLinkRecord {
	return jiraIssueLinkRecord{
		ID:          link.ID.String(),
		LicenseID:   link.LicenseID.String(),
		IssueKey:    link.IssueKey,
		IssueURL:    link.IssueURL,
		Status:      link.Status,
		RenewalDate: cloneTime(link.RenewalDate),
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}
}

func licenseMatches(record licenseRecord, query string) bool {
	fields := []string{
		record.VendorName,
		record.ProductName,
		record.Department,
		record.LicenseKey,
		record.SubscriptionID,
		record.ContractNumber,
		record.PurchaseOrder,
		record.Invoice,
		record.Currency,
		record.Notes,
		record.Type,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(record.ID), query)
}

func vendorMatches(record vendorRecord, query string) bool {
	fields := []string{
		record.Name,
		record.SupportURL,
		record.AccountManager,
		record.Notes,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	for _, contact := range record.Contacts {
		if strings.Contains(strings.ToLower(contact.Name), query) || strings.Contains(strings.ToLower(contact.Email), query) || strings.Contains(strings.ToLower(contact.Phone), query) || strings.Contains(strings.ToLower(contact.Role), query) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(record.ID), query)
}

func licenseSortKey(record licenseRecord) string {
	return strings.ToLower(record.VendorName + "|" + record.ProductName + "|" + record.LicenseKey + "|" + record.ID)
}

func paginate[T any](items []T, limit, offset int) []T {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset >= len(items) {
		return nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]T(nil), items[offset:end]...)
}

func listLicenses(ctx context.Context, svc app.LicenseService) ([]domain.License, error) {
	if svc == nil {
		return nil, nil
	}
	return listAll(ctx, svc.List)
}

func listVendors(ctx context.Context, svc app.VendorService) ([]domain.Vendor, error) {
	if svc == nil {
		return nil, nil
	}
	return listAll(ctx, svc.List)
}

func listProducts(ctx context.Context, svc app.ProductService) ([]domain.Product, error) {
	if svc == nil {
		return nil, nil
	}
	return listAll(ctx, svc.List)
}

func listAssignments(ctx context.Context, svc app.AssignmentService) ([]domain.Assignment, error) {
	if svc == nil {
		return nil, nil
	}
	return listAll(ctx, svc.List)
}

func listAll[T any](ctx context.Context, fn func(context.Context, domain.ListFilter) ([]T, error)) ([]T, error) {
	const pageSize = 500
	var out []T
	for offset := 0; ; offset += pageSize {
		batch, err := fn(ctx, domain.ListFilter{Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return out, nil
}

func normalizeTime(value *time.Time) time.Time {
	if value == nil || value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func parseDecimal(value string) decimal.Decimal {
	if strings.TrimSpace(value) == "" {
		return decimal.Zero
	}
	parsed, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return decimal.Zero
	}
	return parsed
}

func boolPtr(value bool) *bool {
	return &value
}

func promptAsOfSuffix(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return " as of " + strings.TrimSpace(value)
}

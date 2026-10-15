package tosend

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// AdminKeyPrefix is the prefix every admin API key carries.
const AdminKeyPrefix = "tsend_admin_"

// AdminClient is the Admin API client: account-level management of sending
// domains, delivery logs, suppressions, webhooks, sending keys and client
// teams. See https://tosend.com/docs/api/admin-api.
//
// It is deliberately a separate type from Client: admin keys and sending keys
// are different credentials. A sending key cannot manage the account, and an
// admin key can only send when it holds the emails:send scope. Each method
// needs the matching scope on the key; a missing scope returns an *Error whose
// Code() is "insufficient_scope".
type AdminClient struct {
	client    *Client
	accountID int
}

// NewAdmin creates an Admin API client. It returns an error unless apiKey is
// an admin key (it starts with "tsend_admin_"), because pasting a sending key
// is the likeliest mistake and should be caught before any request.
func NewAdmin(apiKey string, opts ...Option) (*AdminClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("tosend: admin API key is required")
	}
	if !strings.HasPrefix(apiKey, AdminKeyPrefix) {
		return nil, errors.New(`tosend: This is not an admin API key. Admin keys start with "tsend_admin_" and are created in the dashboard under API Keys.`)
	}
	return &AdminClient{client: New(apiKey, opts...)}, nil
}

// ForAccount returns a copy of the client that acts on one of your child
// teams. Its requests carry X-Account-Id, so they apply to that team rather
// than to your own account.
func (a *AdminClient) ForAccount(accountID int) *AdminClient {
	return &AdminClient{client: a.client, accountID: accountID}
}

func (a *AdminClient) do(method, path string, query url.Values, body, result interface{}) error {
	var headers map[string]string
	if a.accountID > 0 {
		headers = map[string]string{"X-Account-Id": strconv.Itoa(a.accountID)}
	}
	return a.client.do(method, path, query, headers, body, result)
}

// adminData unwraps the {"data": ...} envelope single-resource endpoints return.
func adminData[T any](a *AdminClient, method, path string, query url.Values, body interface{}) (*T, error) {
	var resp struct {
		Data T `json:"data"`
	}
	if err := a.do(method, path, query, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func adminPage[T any](a *AdminClient, path string, query url.Values) (*Page[T], error) {
	var page Page[T]
	if err := a.do("GET", path, query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// Page is one page of a list endpoint.
type Page[T any] struct {
	Data    []T `json:"data"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// ListParams pages through a list endpoint. Zero values use the API defaults
// (page 1, 25 per page, at most 100).
type ListParams struct {
	Page    int
	PerPage int
}

func (p ListParams) values() url.Values {
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(p.PerPage))
	}
	return q
}

func setIf(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

// ---------------------------------------------------------------------------
// Account
// ---------------------------------------------------------------------------

// AdminInfo is GET /v2/info as seen by an admin key.
type AdminInfo struct {
	Account AdminAccount `json:"account"`
	Key     AdminKeyInfo `json:"key"`
	// Domains is only returned to a key holding domains:read.
	Domains []AdminDomain `json:"domains,omitempty"`
}

// AdminAccount describes the team the request acted on.
type AdminAccount struct {
	ID       int    `json:"id"`
	Status   string `json:"status"`
	PlanType string `json:"plan_type"`
	// CreditBalance is the pooled balance for the whole account, never a
	// child team's own figure. Nil means the account does not consume credits.
	CreditBalance  *int    `json:"credit_balance"`
	CreditsShared  bool    `json:"credits_shared"`
	LimitPerSecond int     `json:"limit_per_second"`
	SESRegion      string  `json:"ses_region"`
	BounceRate     float64 `json:"bounce_rate"`
	ComplaintRate  float64 `json:"complaint_rate"`
}

// AdminKeyInfo describes the admin key that made the request.
type AdminKeyInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
	// ActingOn is the team this request acted on; OwnedBy is the team that
	// owns the key. They differ when X-Account-Id targets a child team.
	ActingOn int      `json:"acting_on"`
	OwnedBy  int      `json:"owned_by"`
	Scopes   []string `json:"scopes"`
}

// GetInfo returns the account the key acts on and the key's own scopes. It
// needs no scope, which makes it the quickest way to check a key works.
func (a *AdminClient) GetInfo() (*AdminInfo, error) {
	var info AdminInfo
	if err := a.do("GET", "/v2/info", nil, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ---------------------------------------------------------------------------
// Sending (requires the emails:send scope)
// ---------------------------------------------------------------------------

// Send sends an email as the targeted team (the key's own, or the one set
// with ForAccount). Needs the emails:send scope.
func (a *AdminClient) Send(req *SendEmailRequest) (*SendEmailResponse, error) {
	var resp SendEmailResponse
	if err := a.do("POST", "/v2/emails", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Batch sends up to 100 emails in one request; same scope and targeting as
// Send.
func (a *AdminClient) Batch(emails []SendEmailRequest) (*BatchEmailResponse, error) {
	var resp BatchEmailResponse
	if err := a.do("POST", "/v2/emails/batch", nil, BatchEmailRequest{Emails: emails}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Domains
// ---------------------------------------------------------------------------

// AdminDomain is a sending domain as the Admin API lists it.
type AdminDomain struct {
	// ID is the domain identifier used across the Admin API.
	ID                 string `json:"id"`
	DomainName         string `json:"domain_name"`
	VerificationStatus string `json:"verification_status"`
	IsDefault          bool   `json:"is_default"`
	SESRegion          string `json:"ses_region"`
	TotalSent          int    `json:"total_sent"`
	TotalBounced       int    `json:"total_bounced"`
	TotalComplained    int    `json:"total_complained"`
	UpdatedAt          string `json:"updated_at"`
}

// CreatedDomain is the domain POST /v2/domains returns.
type CreatedDomain struct {
	ID                 string `json:"id"`
	DomainName         string `json:"domain_name"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt          string `json:"created_at"`
}

// DNSRecord is one record the domain owner must publish.
type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      string `json:"TTL"`
	Priority string `json:"priority"`
}

// DomainDetail is a domain plus the DNS records to publish.
type DomainDetail struct {
	ID                 string  `json:"id"`
	DomainName         string  `json:"domain_name"`
	VerificationStatus string  `json:"verification_status"`
	DMARCPolicy        *string `json:"dmarc_policy"`
	CreatedAt          string  `json:"created_at"`
	DNSRecords         struct {
		// DKIM is unique to the team that owns the domain. Do not reuse it
		// for another domain or team; it will not verify.
		DKIM DNSRecord   `json:"dkim"`
		SPF  []DNSRecord `json:"spf"`
	} `json:"dns_records"`
	// AutoVerification reports whether the automatic 10-minute DNS check
	// still covers this domain.
	AutoVerification struct {
		Active bool   `json:"active"`
		Reason string `json:"reason"`
	} `json:"auto_verification"`
}

// DeletedDomain is the response to DeleteDomain.
type DeletedDomain struct {
	Deleted bool `json:"deleted"`
	// Warning reminds you to remove the domain's DNS records yourself.
	Warning string `json:"warning"`
}

// ListDomains lists the team's domains. Needs domains:read.
func (a *AdminClient) ListDomains(params ListParams) (*Page[AdminDomain], error) {
	return adminPage[AdminDomain](a, "/v2/domains", params.values())
}

// CreateDomain adds a sending domain. Needs domains:write.
//
// The domain starts in draft. There is no verify call: toSend checks public
// DNS every 10 minutes and provisions it once the records resolve. A 409
// means the domain already exists (names are unique across toSend), which a
// retry after a timeout will hit for a domain the first call created.
func (a *AdminClient) CreateDomain(name string) (*CreatedDomain, error) {
	return adminData[CreatedDomain](a, "POST", "/v2/domains", nil, map[string]string{"name": name})
}

// GetDomain returns a domain and the DNS records to publish. Needs
// domains:read. Poll VerificationStatus on a slow interval, not in a loop.
func (a *AdminClient) GetDomain(id string) (*DomainDetail, error) {
	return adminData[DomainDetail](a, "GET", "/v2/domains/"+url.PathEscape(id), nil, nil)
}

// DeleteDomain removes a domain. Needs domains:write. It does not touch the
// customer's DNS zone; records left published can let the domain be
// re-verified elsewhere, so remove them.
func (a *AdminClient) DeleteDomain(id string) (*DeletedDomain, error) {
	var resp DeletedDomain
	if err := a.do("DELETE", "/v2/domains/"+url.PathEscape(id), nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Email logs
// ---------------------------------------------------------------------------

// Recipients decodes the recipient list of a log entry. An entry whose stored
// list cannot be read decodes as empty rather than failing the whole page.
type Recipients []Address

// UnmarshalJSON tolerates a malformed stored recipient list.
func (r *Recipients) UnmarshalJSON(data []byte) error {
	var list []Address
	if err := json.Unmarshal(data, &list); err != nil {
		*r = nil
		return nil
	}
	*r = list
	return nil
}

// EmailLog is one message in the delivery log.
type EmailLog struct {
	// ID is the message_id POST /v2/emails returned.
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	Subject         string     `json:"subject"`
	From            string     `json:"from"`
	FromName        string     `json:"from_name"`
	To              Recipients `json:"to"`
	ReplyTo         string     `json:"reply_to"`
	RecipientsCount int        `json:"recipients_count"`
	Error           string     `json:"error"`
	HasAttachments  bool       `json:"has_attachments"`
	SentAt          string     `json:"sent_at"`
	CreatedAt       string     `json:"created_at"`
}

// ListEmailsParams filters the delivery log.
type ListEmailsParams struct {
	Page    int
	PerPage int
	// Status is pending, sent, bounced, complained, suppressed or failed.
	Status string
	// Domain is a domain name or domain id.
	Domain string
	// From and To are ISO 8601 date bounds.
	From string
	To   string
	// Search matches subject, from address or recipient. It counts against
	// the write rate limit.
	Search string
}

// ListEmails lists the team's delivery log. Needs emails:read.
func (a *AdminClient) ListEmails(params ListEmailsParams) (*Page[EmailLog], error) {
	q := ListParams{Page: params.Page, PerPage: params.PerPage}.values()
	setIf(q, "status", params.Status)
	setIf(q, "domain", params.Domain)
	setIf(q, "from", params.From)
	setIf(q, "to", params.To)
	setIf(q, "search", params.Search)
	return adminPage[EmailLog](a, "/v2/emails", q)
}

// GetEmail looks up one message by the id POST /v2/emails returned. Needs
// emails:read.
func (a *AdminClient) GetEmail(messageID string) (*EmailLog, error) {
	return adminData[EmailLog](a, "GET", "/v2/emails/"+url.PathEscape(messageID), nil, nil)
}

// ---------------------------------------------------------------------------
// Suppressions
// ---------------------------------------------------------------------------

// Suppression is a suppressed address.
type Suppression struct {
	Email           string `json:"email"`
	BounceSubType   string `json:"bounce_sub_type"`
	BounceReason    string `json:"bounce_reason"`
	HardBouncedAt   string `json:"hard_bounced_at"`
	LastComplaintAt string `json:"last_complaint_at"`
	ComplaintCount  int    `json:"complaint_count"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ListSuppressionsParams filters the suppression list.
type ListSuppressionsParams struct {
	Page    int
	PerPage int
	// Search counts against the write rate limit.
	Search string
	// Status is bounced, complained or suppressed.
	Status string
}

// SuppressionRemoval is the response to DeleteSuppression.
type SuppressionRemoval struct {
	Deleted        bool   `json:"deleted"`
	Message        string `json:"message"`
	UsedToday      int    `json:"used_today"`
	RemainingToday int    `json:"remaining_today"`
	DailyLimit     int    `json:"daily_limit"`
}

// ListSuppressions lists the team's suppressed addresses. Needs
// suppressions:read.
func (a *AdminClient) ListSuppressions(params ListSuppressionsParams) (*Page[Suppression], error) {
	q := ListParams{Page: params.Page, PerPage: params.PerPage}.values()
	setIf(q, "search", params.Search)
	setIf(q, "status", params.Status)
	return adminPage[Suppression](a, "/v2/suppressions", q)
}

// DeleteSuppression reactivates an address. Needs suppressions:write.
// Limited to 500 per day per account, shared with the dashboard.
func (a *AdminClient) DeleteSuppression(email string) (*SuppressionRemoval, error) {
	var resp SuppressionRemoval
	if err := a.do("DELETE", "/v2/suppressions", url.Values{"email": {email}}, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Webhooks
// ---------------------------------------------------------------------------

// Webhook events toSend emits. Delivery, open and click events do not exist.
const (
	WebhookEventBounced   = "bounced"
	WebhookEventComplaint = "complaint"
)

// Webhook is a webhook subscription.
type Webhook struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	URL            string   `json:"url"`
	Events         []string `json:"events"`
	Status         string   `json:"status"`
	IncludeMessage bool     `json:"include_message"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	// Secret signs every delivery. It is set only on the webhook
	// CreateWebhook returns; store it then.
	Secret string `json:"secret,omitempty"`
}

// CreateWebhookParams creates a webhook. URL and Events are required.
type CreateWebhookParams struct {
	Title          string   `json:"title,omitempty"`
	URL            string   `json:"url"`
	Events         []string `json:"events"`
	IncludeMessage bool     `json:"include_message,omitempty"`
}

// UpdateWebhookParams changes a webhook. Nil fields are left unchanged.
type UpdateWebhookParams struct {
	Title  *string  `json:"title,omitempty"`
	URL    *string  `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	// Status is "active" or "inactive" (paused).
	Status         *string `json:"status,omitempty"`
	IncludeMessage *bool   `json:"include_message,omitempty"`
}

// ListWebhooks lists the team's webhooks. Needs webhooks:read.
func (a *AdminClient) ListWebhooks(params ListParams) (*Page[Webhook], error) {
	return adminPage[Webhook](a, "/v2/webhooks", params.values())
}

// CreateWebhook creates a webhook. Needs webhooks:write. The returned
// webhook carries its signing Secret, which toSend generates and the API
// returns only here.
func (a *AdminClient) CreateWebhook(params CreateWebhookParams) (*Webhook, error) {
	return adminData[Webhook](a, "POST", "/v2/webhooks", nil, params)
}

// UpdateWebhook changes a webhook. Needs webhooks:write.
func (a *AdminClient) UpdateWebhook(id int, params UpdateWebhookParams) (*Webhook, error) {
	return adminData[Webhook](a, "PUT", "/v2/webhooks/"+strconv.Itoa(id), nil, params)
}

// DeleteWebhook deletes a webhook. Needs webhooks:write.
func (a *AdminClient) DeleteWebhook(id int) error {
	return a.do("DELETE", "/v2/webhooks/"+strconv.Itoa(id), nil, nil, nil)
}

// ---------------------------------------------------------------------------
// Sending API keys
// ---------------------------------------------------------------------------

// SendingKey is a sending key, masked.
type SendingKey struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// Key is masked. The full key is returned only by CreateSendingKey.
	Key    string `json:"key"`
	Status string `json:"status"`
	// Domain is the domain name the key is restricted to, or nil for all.
	Domain    *string `json:"domain"`
	CreatedAt string  `json:"created_at"`
}

// CreatedSendingKey is a newly minted sending key.
type CreatedSendingKey struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// APIKey is shown once. Store it now; it cannot be retrieved later.
	APIKey    string `json:"api_key"`
	CreatedAt string `json:"created_at"`
}

// ListSendingKeys lists the team's sending keys. Needs api_keys:read.
func (a *AdminClient) ListSendingKeys(params ListParams) (*Page[SendingKey], error) {
	return adminPage[SendingKey](a, "/v2/api-keys", params.values())
}

// CreateSendingKey mints a sending key for this team, which is how you give
// each client their own. Pass a domain id to restrict it to one domain, or ""
// for all domains on the team. Needs api_keys:write and emails:send.
func (a *AdminClient) CreateSendingKey(name, domain string) (*CreatedSendingKey, error) {
	body := map[string]string{"name": name}
	if domain != "" {
		body["domain"] = domain
	}
	return adminData[CreatedSendingKey](a, "POST", "/v2/api-keys", nil, body)
}

// DeleteSendingKey revokes a sending key immediately. Needs api_keys:write.
func (a *AdminClient) DeleteSendingKey(id int) error {
	return a.do("DELETE", "/v2/api-keys/"+strconv.Itoa(id), nil, nil, nil)
}

// ---------------------------------------------------------------------------
// Teams
// ---------------------------------------------------------------------------

// Team is a child team.
type Team struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	// SesRegion is where the team sends from (RegionUS or RegionEU).
	SesRegion string `json:"ses_region,omitempty"`
}

// Sending regions for CreateTeamInRegion. A team's region is fixed for life.
const (
	RegionUS = "us-east-1"
	RegionEU = "eu-central-1"
)

// TeamList is your account and its child teams.
type TeamList struct {
	Account struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		// CreditBalance is the pooled balance for the whole account. Child
		// teams draw on it and have no balance of their own.
		CreditBalance *int `json:"credit_balance"`
	} `json:"account"`
	Teams         []Team `json:"teams"`
	CreditsShared bool   `json:"credits_shared"`
}

// UpdateTeamParams changes a team. Empty fields are left unchanged.
type UpdateTeamParams struct {
	Name string `json:"name,omitempty"`
	// Status is "active" or "suspended". A team toSend suspended cannot be
	// reactivated here (403, Code() "suspended_by_platform").
	Status string `json:"status,omitempty"`
}

// ListTeams lists the key owner's account and its child teams, whatever
// ForAccount targets. Needs teams:read.
func (a *AdminClient) ListTeams() (*TeamList, error) {
	return adminData[TeamList](a, "GET", "/v2/teams", nil, nil)
}

// CreateTeam creates a child team (name 2 to 100 characters). Only the
// parent account can create teams, so do not call it on a ForAccount client.
// Needs teams:write.
func (a *AdminClient) CreateTeam(name string) (*Team, error) {
	return adminData[Team](a, "POST", "/v2/teams", nil, map[string]string{"name": name})
}

// CreateTeamInRegion is CreateTeam with a sending region (RegionUS or
// RegionEU). The region cannot be changed once the team exists.
func (a *AdminClient) CreateTeamInRegion(name, region string) (*Team, error) {
	return adminData[Team](a, "POST", "/v2/teams", nil, map[string]string{"name": name, "ses_region": region})
}

// UpdateTeam renames a child team or changes its status. Needs teams:write.
func (a *AdminClient) UpdateTeam(id int, params UpdateTeamParams) (*Team, error) {
	return adminData[Team](a, "PUT", "/v2/teams/"+strconv.Itoa(id), nil, params)
}

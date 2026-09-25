package tosend

import "fmt"

// Address represents an email address with optional name.
type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Attachment represents an email attachment.
type Attachment struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"` // Base64 encoded
}

// SendEmailRequest represents the request to send a single email.
type SendEmailRequest struct {
	From        Address           `json:"from"`
	To          []Address         `json:"to"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	CC          []Address         `json:"cc,omitempty"`
	BCC         []Address         `json:"bcc,omitempty"`
	ReplyTo     *Address          `json:"reply_to,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
}

// SendEmailResponse represents the response from sending an email.
type SendEmailResponse struct {
	MessageID string `json:"message_id"`
}

// Send sends a single email.
func (c *Client) Send(req *SendEmailRequest) (*SendEmailResponse, error) {
	var resp SendEmailResponse
	if err := c.request("POST", "/v2/emails", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchEmailResult represents the result of a single email in a batch.
type BatchEmailResult struct {
	Status     string         `json:"status"`
	MessageID  string         `json:"message_id,omitempty"`
	StatusCode int            `json:"status_code,omitempty"`
	ErrorType  string         `json:"error_type,omitempty"`
	Message    string         `json:"message,omitempty"`
	Errors     map[string]any `json:"errors,omitempty"`
}

// BatchEmailRequest represents the request to send multiple emails.
type BatchEmailRequest struct {
	Emails []SendEmailRequest `json:"emails"`
}

// BatchEmailResponse represents the response from sending batch emails.
type BatchEmailResponse struct {
	Results []BatchEmailResult `json:"results"`
}

// Batch sends multiple emails in a single request.
func (c *Client) Batch(emails []SendEmailRequest) (*BatchEmailResponse, error) {
	req := BatchEmailRequest{Emails: emails}
	var resp BatchEmailResponse
	if err := c.request("POST", "/v2/emails/batch", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Flag decodes a JSON boolean or a 0/1 number. The sending API's
// /v2/info reports is_default as an integer.
type Flag bool

// UnmarshalJSON accepts true, false, 0, 1 and null.
func (f *Flag) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true", "1":
		*f = true
	case "false", "0", "null":
		*f = false
	default:
		return fmt.Errorf("tosend: cannot decode %s as a flag", data)
	}
	return nil
}

// Domain represents a sending domain on the account.
type Domain struct {
	ID                 int    `json:"id"`
	DomainName         string `json:"domain_name"`
	VerificationStatus string `json:"verification_status"`
	IsDefault          Flag   `json:"is_default"`
	TotalSent          int    `json:"total_sent"`
	TotalBounced       int    `json:"total_bounced"`
	TotalComplained    int    `json:"total_complained"`
}

// Account represents account information.
type Account struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	// PlanType is the account's plan, for example "free" or "pro".
	PlanType string `json:"plan_type"`
	// CreditBalance is nil for an account that does not consume credits.
	CreditBalance     *int    `json:"credit_balance"`
	MonthlyEmailLimit int     `json:"monthly_email_limit"`
	LimitPerSecond    int     `json:"limit_per_second"`
	SESRegion         string  `json:"ses_region"`
	BounceRate        float64 `json:"bounce_rate"`    // fraction between 0 and 1
	ComplaintRate     float64 `json:"complaint_rate"` // fraction between 0 and 1
	SESStatus         string  `json:"ses_status"`
}

// AccountInfo represents the account info response.
type AccountInfo struct {
	Account Account  `json:"account"`
	Domains []Domain `json:"domains"`
}

// GetAccountInfo retrieves account information and domains.
func (c *Client) GetAccountInfo() (*AccountInfo, error) {
	var resp AccountInfo
	if err := c.request("GET", "/v2/info", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

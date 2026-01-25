package tosend

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
	From        Address            `json:"from"`
	To          []Address          `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html,omitempty"`
	Text        string             `json:"text,omitempty"`
	CC          []Address          `json:"cc,omitempty"`
	BCC         []Address          `json:"bcc,omitempty"`
	ReplyTo     *Address           `json:"reply_to,omitempty"`
	Headers     map[string]string  `json:"headers,omitempty"`
	Attachments []Attachment       `json:"attachments,omitempty"`
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
	Status    string                       `json:"status"`
	MessageID string                       `json:"message_id,omitempty"`
	Message   string                       `json:"message,omitempty"`
	Errors    map[string]map[string]string `json:"errors,omitempty"`
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

// Domain represents a verified domain.
type Domain struct {
	DomainName         string `json:"domain_name"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt          string `json:"created_at"`
}

// Account represents account information.
type Account struct {
	Title               string `json:"title"`
	PlanType            string `json:"plan_type"`
	Status              string `json:"status"`
	EmailsUsageThisMonth int   `json:"emails_usage_this_month"`
	EmailsSentLast24Hrs  int   `json:"emails_sent_last_24hrs"`
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

package tosend

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type captured struct {
	method, path, escapedPath, rawQuery, accountID, auth, body string
	query                                                      map[string][]string
}

func server(t *testing.T, status int, body string, got *captured) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		*got = captured{
			body:        string(raw),
			method:      r.Method,
			path:        r.URL.Path,
			escapedPath: r.URL.EscapedPath(),
			rawQuery:    r.URL.RawQuery,
			accountID:   r.Header.Get("X-Account-Id"),
			auth:        r.Header.Get("Authorization"),
			query:       r.URL.Query(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func admin(t *testing.T, url string) *AdminClient {
	t.Helper()
	a, err := NewAdmin("tsend_admin_abc123", WithBaseURL(url))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestNewAdminRejectsSendingKeys(t *testing.T) {
	if _, err := NewAdmin("tsend_0a1b2c3d4e5f"); err == nil || !strings.Contains(err.Error(), "not an admin API key") {
		t.Fatalf("expected a prefix error, got %v", err)
	}
	if _, err := NewAdmin("   "); err == nil {
		t.Fatal("expected an error for an empty key")
	}
	if _, err := NewAdmin(" tsend_admin_abc "); err != nil {
		t.Fatalf("admin key rejected: %v", err)
	}
}

func TestForAccountSendsHeaderAndBaseDoesNot(t *testing.T) {
	var got captured
	srv := server(t, 200, `{"data":{"account":{"id":1},"teams":[],"credits_shared":true}}`, &got)
	a := admin(t, srv.URL)

	if _, err := a.ListTeams(); err != nil {
		t.Fatal(err)
	}
	if got.accountID != "" {
		t.Fatalf("base client sent X-Account-Id %q", got.accountID)
	}
	if got.auth != "Bearer tsend_admin_abc123" {
		t.Fatalf("auth header %q", got.auth)
	}

	if _, err := a.ForAccount(4203).ListTeams(); err != nil {
		t.Fatal(err)
	}
	if got.accountID != "4203" {
		t.Fatalf("ForAccount sent X-Account-Id %q", got.accountID)
	}

	// ForAccount returns a copy; the original stays untargeted.
	if _, err := a.ListTeams(); err != nil {
		t.Fatal(err)
	}
	if got.accountID != "" {
		t.Fatalf("ForAccount mutated the base client: %q", got.accountID)
	}
}

func TestFlatAdminErrorDecodes(t *testing.T) {
	var got captured
	srv := server(t, 403, `{"status_code":403,"message":"This admin key does not have the domains:read scope.","errors":{"code":"insufficient_scope","required_scope":"domains:read"}}`, &got)

	_, err := admin(t, srv.URL).ListDomains(ListParams{})
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T %v", err, err)
	}
	if apiErr.StatusCode != 403 || apiErr.Code() != "insufficient_scope" {
		t.Fatalf("got status %d code %q", apiErr.StatusCode, apiErr.Code())
	}
	if apiErr.Errors["required_scope"] != "domains:read" {
		t.Fatalf("required_scope lost: %v", apiErr.Errors)
	}
	if !strings.Contains(apiErr.Message, "domains:read scope") {
		t.Fatalf("message lost: %q", apiErr.Message)
	}
}

func TestNestedSendingErrorDecodes(t *testing.T) {
	var got captured
	srv := server(t, 422, `{"status_code":422,"error_type":"validation_error","message":"Subject is required.","errors":{"subject":{"required":"Subject is required."}}}`, &got)

	_, err := New("tsend_key", WithBaseURL(srv.URL)).Send(&SendEmailRequest{})
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T %v", err, err)
	}
	if apiErr.ErrorType != "validation_error" || apiErr.Code() != "" {
		t.Fatalf("got type %q code %q", apiErr.ErrorType, apiErr.Code())
	}
	field, ok := apiErr.Errors["subject"].(map[string]any)
	if !ok || field["required"] != "Subject is required." {
		t.Fatalf("nested errors lost: %v", apiErr.Errors)
	}
}

func TestErrorWithoutStatusCodeUsesHTTPStatus(t *testing.T) {
	var got captured
	srv := server(t, 503, `<html>Bad gateway</html>`, &got)

	_, err := admin(t, srv.URL).GetInfo()
	apiErr, ok := err.(*Error)
	if !ok || apiErr.StatusCode != 503 || !strings.Contains(apiErr.Message, "Bad gateway") {
		t.Fatalf("got %T %+v", err, err)
	}
}

func TestSendingGetAccountInfoDecodesRealShape(t *testing.T) {
	var got captured
	srv := server(t, 200, `{"account":{"id":142,"status":"active","plan_type":"pro","credit_balance":48750,"monthly_email_limit":100000,"limit_per_second":14,"ses_region":"us-east-1","bounce_rate":0.012,"complaint_rate":0.0003,"ses_status":"healthy"},"domains":[{"id":88,"domain_name":"acme.com","verification_status":"verified","is_default":1,"total_sent":24189,"total_bounced":212,"total_complained":7}]}`, &got)

	info, err := New("tsend_key", WithBaseURL(srv.URL)).GetAccountInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Account.ID != 142 || info.Account.CreditBalance == nil || *info.Account.CreditBalance != 48750 ||
		info.Account.LimitPerSecond != 14 || info.Account.BounceRate != 0.012 || info.Account.SESStatus != "healthy" {
		t.Fatalf("account decoded wrong: %+v", info.Account)
	}
	d := info.Domains[0]
	if d.ID != 88 || !bool(d.IsDefault) || d.TotalSent != 24189 {
		t.Fatalf("domain decoded wrong: %+v", d)
	}
}

func TestAdminGetInfoDecodes(t *testing.T) {
	var got captured
	srv := server(t, 200, `{"account":{"id":4182,"status":"active","credit_balance":null,"credits_shared":false},"key":{"id":91,"title":"Provisioning","type":"admin","acting_on":4182,"owned_by":4182,"scopes":["domains:read","domains:write"]},"domains":[{"id":"9f2c","domain_name":"client.example.com","verification_status":"verified","is_default":false}]}`, &got)

	info, err := admin(t, srv.URL).GetInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Account.CreditBalance != nil || info.Key.OwnedBy != 4182 || len(info.Key.Scopes) != 2 || info.Domains[0].ID != "9f2c" {
		t.Fatalf("decoded wrong: %+v", info)
	}
	if got.path != "/v2/info" {
		t.Fatalf("path %q", got.path)
	}
}

func TestListSendingKeysPages(t *testing.T) {
	var got captured
	srv := server(t, 200, `{"total":30,"page":2,"per_page":10,"data":[{"id":1,"title":"A","key":"tsend_0a...e8f9","status":"active","domain":null,"created_at":"x"}]}`, &got)

	page, err := admin(t, srv.URL).ListSendingKeys(ListParams{Page: 2, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v2/api-keys" || got.query["page"][0] != "2" || got.query["per_page"][0] != "10" {
		t.Fatalf("request %+v", got)
	}
	if page.Total != 30 || page.Page != 2 || len(page.Data) != 1 || page.Data[0].Domain != nil {
		t.Fatalf("page decoded wrong: %+v", page)
	}
}

func TestDeleteSuppressionEncodesEmail(t *testing.T) {
	var got captured
	srv := server(t, 200, `{"deleted":true,"message":"ok","used_today":1,"remaining_today":499,"daily_limit":500}`, &got)

	res, err := admin(t, srv.URL).DeleteSuppression("a+b@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "DELETE" || got.path != "/v2/suppressions" || got.query["email"][0] != "a+b@example.com" {
		t.Fatalf("request %+v", got)
	}
	if !strings.Contains(got.rawQuery, "a%2Bb%40example.com") {
		t.Fatalf("email not encoded: %q", got.rawQuery)
	}
	if res.RemainingToday != 499 || res.DailyLimit != 500 {
		t.Fatalf("decoded wrong: %+v", res)
	}
}

func TestPathIDsAreEscaped(t *testing.T) {
	var got captured
	srv := server(t, 200, `{"data":{"id":"x"}}`, &got)

	if _, err := admin(t, srv.URL).GetEmail("msg/../1"); err != nil {
		t.Fatal(err)
	}
	if got.escapedPath != "/v2/emails/msg%2F..%2F1" {
		t.Fatalf("id not escaped: %q", got.escapedPath)
	}
}

func TestUpdateWebhookOmitsUnsetFields(t *testing.T) {
	status := "inactive"
	body, err := json.Marshal(UpdateWebhookParams{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"status":"inactive"}` {
		t.Fatalf("got %s", body)
	}
}

func TestBatchResultFlatErrors(t *testing.T) {
	var got captured
	srv := server(t, 207, `{"results":[{"status":"error","status_code":422,"error_type":"validation_error","message":"Validation failed","errors":{"to":{"required":"x"}}},{"status":"success","message_id":"m1"}]}`, &got)

	// 207 is below 400, so it decodes as a response rather than an error.
	resp, err := New("tsend_key", WithBaseURL(srv.URL)).Batch([]SendEmailRequest{{}, {}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Results[0].StatusCode != 422 || resp.Results[0].Errors["to"] == nil || resp.Results[1].MessageID != "m1" {
		t.Fatalf("decoded wrong: %+v", resp.Results)
	}
}

func TestCreateTeamInRegionSendsRegion(t *testing.T) {
	var got captured
	srv := server(t, 201, `{"data":{"id":8,"name":"Clients EU","status":"active","ses_region":"eu-central-1"}}`, &got)

	team, err := admin(t, srv.URL).CreateTeamInRegion("Clients EU", RegionEU)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v2/teams" || got.body != `{"name":"Clients EU","ses_region":"eu-central-1"}` {
		t.Fatalf("request = %s %s %s", got.method, got.path, got.body)
	}
	if team.SesRegion != "eu-central-1" {
		t.Fatalf("SesRegion = %q", team.SesRegion)
	}

	if _, err := admin(t, srv.URL).CreateTeam("Clients"); err != nil {
		t.Fatal(err)
	}
	if got.body != `{"name":"Clients"}` {
		t.Fatalf("CreateTeam body = %s", got.body)
	}
}

func TestAttachmentContentIDOmittedWhenEmpty(t *testing.T) {
	inline, _ := json.Marshal(Attachment{Type: "image/png", Name: "logo.png", Content: "cG5n", ContentID: "logo@example.com"})
	plain, _ := json.Marshal(Attachment{Type: "application/pdf", Name: "a.pdf", Content: "cGRm"})
	if string(inline) != `{"type":"image/png","name":"logo.png","content":"cG5n","content_id":"logo@example.com"}` {
		t.Fatalf("inline = %s", inline)
	}
	if string(plain) != `{"type":"application/pdf","name":"a.pdf","content":"cGRm"}` {
		t.Fatalf("plain = %s", plain)
	}
}
